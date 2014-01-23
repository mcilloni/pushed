package backend

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"
)

const (
	GcmDefaultMaxHttpConns       = 5
	GcmDefaultMaxSleepBeforeFail = 8 * time.Second
	gcmRequestUrl                = "https://android.googleapis.com/gcm/send"
)

var (
	GcmAuthError            = errors.New("GCM: The given GCM authkey is invalid. Check it twice.")
	GcmInternalServerError  = errors.New("GCM: Internal server error.")
	GcmMessageTooLargeError = errors.New("GCM: Message is bigger than 4 KiBs (4096 bytes)")
	GcmNoRegIdForUser       = errors.New("GCM: No Registration IDs for given user")
	GcmTimeoutError         = errors.New("GCM: Timeout or server unavailable.")
	GcmUnknownStatusError   = errors.New("GCM: Unknown status. Fatal.")
	GcmWontTryAgain         = errors.New("GCM: Connector has given up with message delivery")
)

type GcmConfig struct {
	ApiKey       string
	MaxTcpConns  int
	MaxRetryTime time.Duration
}

type gcm struct {
	apiKey   string
	client   *http.Client
	maxSleep time.Duration
}

type gcmPayload struct {
	RegIds []string          `json:"registration_ids"`
	Data   map[string]string `json:"data"`
}

func (db *db) gcmInitStmt() (e error) {

	c := db.conn

	db.gcmRegAdd, e = c.Prepare("INSERT INTO GCM VALUES ($1,$2)")

	if e != nil {
		return
	}

	db.gcmRegDel, e = c.Prepare("DELETE FROM GCM WHERE REGID = $1")

	if e != nil {
		return
	}

	db.gcmRegFetch, e = c.Prepare("SELECT REGID FROM GCM WHERE USERID = $1")

	if e != nil {
		return
	}

	db.gcmUpdateReg, e = c.Prepare("UPDATE GCM SET REGID = $2 WHERE REGID = $1")

	return

}

func (db *db) gcmAddRegistrationId(id int64, regid string) error {
	log.Printf("adding GCM RegId for %d", id)

	_, e := db.gcmRegAdd.Exec(id, regid)

	return e
}

func (db *db) gcmCloseStmt() (e error) {

	if e = db.gcmRegAdd.Close(); e != nil {
		return
	}

	if e = db.gcmRegDel.Close(); e != nil {
		return
	}

	if e = db.gcmRegFetch.Close(); e != nil {
		return
	}

	return nil

}

func (db *db) gcmDeleteRegistrationId(regid string) error {

	_, e := db.gcmRegDel.Exec(regid)

	return e
}

func (db *db) gcmGetRegistrationIdsForId(id int64) ([]string, error) {

	rows, e := db.gcmRegFetch.Query(id)

	if e != nil {
		return nil, e
	}

	ids := make([]string, 0, 10)

	var regid string

	for rows.Next() {
		if e = rows.Scan(&regid); e != nil {
			return nil, e
		}

		ids = append(ids, regid)
	}

	if e = rows.Err(); e != nil {
		return nil, e
	}

	if len(ids) == 0 {
		return nil, GcmNoRegIdForUser
	}

	return ids, nil

}

func (db *db) gcmInitTable() error {

	_, e := db.conn.Exec("CREATE TABLE GCM (USERID BIGINT REFERENCES USERS ON DELETE CASCADE, REGID CHARACTER VARYING, PRIMARY KEY (USERID,REGID))")

	if e != nil {
		return e
	}

	log.Println("Creating triggers on GCM table...")

	_, e = db.conn.Exec("CREATE FUNCTION CHECKTEN() RETURNS TRIGGER AS $$ BEGIN IF((SELECT COUNT(REGID) FROM GCM WHERE USERID = NEW.USERID) >= 10) THEN RAISE EXCEPTION 'Already 10 Registration IDs for this user'; END IF; RETURN NEW; END $$ LANGUAGE plpgsql")

	if e != nil {
		return e
	}

	_, e = db.conn.Exec("CREATE TRIGGER CHECKTEN BEFORE INSERT ON GCM FOR EACH ROW EXECUTE PROCEDURE CHECKTEN()")

	return e
}

func (db *db) gcmUpdateRegId(oldId, newId string) error {

	result, e := db.gcmUpdateReg.Exec(oldId, newId)

	if e != nil {
		return e
	}

	rowsAffected, e := result.RowsAffected()

	if e != nil {
		return e
	}

	if rowsAffected > 1 {
		log.Panic("Database inconsistency found (regid found twice or more)")
	}

	return nil

}

func newGcm(config *GcmConfig) *gcm {

    if config.MaxTcpConns == 0 {
        config.MaxTcpConns = GcmDefaultMaxHttpConns
    }

    if config.MaxRetryTime == 0 {
        config.MaxRetryTime = GcmDefaultMaxSleepBeforeFail
    }

	return &gcm{
		apiKey: "key=" + config.ApiKey,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: config.MaxTcpConns,
			},
		},
		maxSleep: config.MaxRetryTime,
	}

}

type gcmOpData struct {
	Delay    time.Duration
	Data     *gcmPayload
	Response *http.Response
}

func (gcm *gcm) Push(user int64, message Message) error {

	ids, e := globalDb.gcmGetRegistrationIdsForId(user)

	if e != nil {
		return e
	}

	return gcm.regidsPush(ids, message)

}

func (gcm *gcm) Register(user int64, deviceTargetId string) error {

	return globalDb.gcmAddRegistrationId(user, deviceTargetId)

}

func (gcm *gcm) Unregister(deviceTargetId string) error {

	return globalDb.gcmDeleteRegistrationId(deviceTargetId)

}

func (gcm *gcm) expRetry(opData *gcmOpData) error {

	if opData.Delay > gcm.maxSleep {
		return GcmWontTryAgain
	}

    sleep := opData.Delay

    if 2*opData.Delay > gcm.maxSleep {
        sleep = gcm.maxSleep
    }

	time.Sleep(sleep)

	return gcm.payloadPush(opData.Data, 2*opData.Delay)

}

func (gcm *gcm) evalResponse(opData *gcmOpData) error {

	res := opData.Response

	defer res.Body.Close()

	switch {
	case res.StatusCode == 200:
		break //all good, all right, but who knows what we could expect from body?

	case res.StatusCode == 400:
		log.Panic("Google rejected our JSON - this is a critical bug in GCM connector. Fix the bug and retry")

	case res.StatusCode == 401:
		return GcmAuthError

	case res.StatusCode == 500:
		log.Println("GCM internal server error, beginning exponential retry")

		if e := gcm.expRetry(opData); e != GcmWontTryAgain {
			return e
		}

		return GcmInternalServerError

	case res.StatusCode >= 501 && res.StatusCode <= 599:
		log.Println("GCM timeout, beginning exponential retry")

		if e := gcm.expRetry(opData); e != GcmWontTryAgain {
			return e
		}

		return GcmTimeoutError

	default:
		return GcmUnknownStatusError
	}

	return gcm.responseBodyParse(opData)

}

func (gcm *gcm) payloadPush(payload *gcmPayload, retryTime time.Duration) error {

	jsonData, e := json.Marshal(payload.Data)

	if e != nil {
		return e
	}

	if len(jsonData) > 4096 {
		return GcmMessageTooLargeError
	}

	jsonPayload, e := json.Marshal(payload)

	if e != nil {
		return e
	}

	req, e := http.NewRequest("POST", gcmRequestUrl, bytes.NewReader(jsonPayload))

	if e != nil {
		return e

	}

	req.Header.Add("Authorization", gcm.apiKey)

	res, e := gcm.client.Do(req)

	if e != nil {
		return e
	}

	opData := &gcmOpData{
		Delay:    retryTime,
		Response: res,
		Data:     payload,
	}

	return gcm.evalResponse(opData)

}

func (gcm *gcm) regidsPush(regids []string, data Message) error {

	if regids == nil {
		return errors.New("Empty regids array")
	}

	gcmP := &gcmPayload{
		RegIds: regids,
		Data:   data,
	}

	return gcm.payloadPush(gcmP, time.Second)

}

type gcmResult struct {
	MessageId string `json:"message_id"`
	CanonId   string `json:"registration_id"`
	Error     string `json:"error"`
}

type gcmResponse struct {
	MulticastId  uint        `json:"multicast_id"`
	Success      uint        `json:"success"`
	Failure      uint        `json:"failure"`
	CanonicalIds uint        `json:"canonical_ids"`
	Results      []gcmResult `json:"results"`
}

func (gcm *gcm) responseBodyParse(opData *gcmOpData) error {

	var response gcmResponse

	e := json.NewDecoder(opData.Response.Body).Decode(&response)

	if e != nil {
		log.Panicf("Received invalid JSON from Google. Report this. Error: %s", e.Error())
	}

	if response.Failure|response.CanonicalIds == 0 {
		return nil //all good, no weird suprises
	}

	//Getting in trouble here...

	if len(response.Results) != len(opData.Data.RegIds) {
		log.Panicf("Malformed response from Google, sent %d registration_ids, recv %d results", len(opData.Data.RegIds), len(response.Results))
	}

	for i, regid := range opData.Data.RegIds {
		if e := gcm.responseEvalLine(regid, &response.Results[i], opData); e != nil {
			return e
		}
	}

	return nil

}

func (gcm *gcm) responseEvalLine(regid string, result *gcmResult, opData *gcmOpData) error {

	if result.MessageId != "" { //all went well, check if a canonical id is given... (http://developer.android.com/google/gcm/adv.html#canonical)
		if result.CanonId != "" {
			return globalDb.gcmUpdateRegId(regid, result.CanonId) //update, than we're good
		}
		return nil //all good, nothing to do
	}

	//Oh, noes, error parsing.
	//Thanks google for shitty docs, anyway.

	if result.Error == "" {
		log.Panic("Empty response from GCM for regid, protocol changed or broken API")
	}

	switch result.Error {
	case "NotRegistered": //User has removed the application
		globalDb.gcmDeleteRegistrationId(regid)
		break
	case "MissingRegistration": //This cannot happen, we always check for regids before sending!
		log.Panic("connector broken, MissingRegistration found")
	case "InvalidRegistration", "MismatchSenderId": //Malformed regid. Probably broken registration or somebody messed with the client. Lets delete it and log it
		globalDb.gcmDeleteRegistrationId(regid)
		log.Printf("GCM RegID %s has been rejected from server with %s and has been deleted.", regid, result.Error)
		break
	case "MessageTooBig":
		log.Panic("connector broken, a message with data bigger than 4KiB has been allowed")
	case "InvalidDataKey":
		log.Printf("A message has been refused from GCM because of an InvalidDataKey in payload")
		break
	case "InvalidTtl":
		log.Panic("This connector has no support to ttl, so this will never happen")
	case "InvalidPackageName":
		log.Println("InvalidPackageName (??)")
		break
	case "InternalServerError":

		if e := gcm.expRetry(opData); e != GcmWontTryAgain {
			return e
		}

		return GcmInternalServerError

	case "Unavailable":

		if e := gcm.expRetry(opData); e != GcmWontTryAgain {
			return e
		}

		return GcmTimeoutError

	default:
		log.Printf("GCM unknown error in response body: %s", result.Error)
		break
	}

	return nil

}
