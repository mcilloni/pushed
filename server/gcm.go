package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	gcmRequestUrl = "https://android.googleapis.com/gcm/send"
)

var (
	GcmAuthError           = errors.New("GCM: The given GCM authkey is invalid. Check it twice.")
	GcmInternalServerError = errors.New("GCM: Internal server error.")
	GcmTimeoutError        = errors.New("GCM: Timeout.")
	GcmUnknownStatusError  = errors.New("GCM: Unknown status. Fatal.")
	GcmWontTryAgain        = errors.New("GCM: Connector has given up with message delivery")
)

type gcm struct {
	apiKey string
	client *http.Client
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

	db.gcmRegDel, e = c.Prepare("DELETE FROM GCM WHERE USERID = $1 AND REGID = $2")

	if e != nil {
		return
	}

	db.gcmRegFetch, e = c.Prepare("SELECT REGID FROM GCM WHERE USERID = $1")

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

func (db *db) gcmDeleteRegistrationId(id int64, regid string) error {
	log.Printf("adding GCM RegId for %d", id)

	_, e := db.gcmRegDel.Exec(id, regid)

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

func newGcm(apiKey string, maxTcpConns int) *gcm {

	return &gcm{
		apiKey: "key=" + apiKey,
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: maxTcpConns,
			},
		},
	}

}

func (gcm *gcm) bodyParse(res *http.Response, retrySec time.Duration) error {

	type gcmResult struct {
		MessageId      string `json:"message_id"`
		RegistrationId string `json:"registration_id"`
		Error          string `json:"error"`
	}

	type gcmResponse struct {
		MulticastId  uint `json:"multicast_id"`
		Success      uint `json:"success"`
		Failure      uint `json:"failure"`
		CanonicalIds uint `json:"canonical_ids"`
		Results      []gcmResult
	}

    var response gcmResponse

	json.NewDecoder(res.Body).Decode(&response)

    if response.Failure | response.CanonicalIds == 0 {
        return nil //all good, no weird suprises
    }

    //Getting in trouble here...

    

}

func (gcm *gcm) expRetry(res *http.Response, retrySec time.Duration, data []byte) error {

	time.Sleep(retrySec)

}

func (gcm *gcm) evalResponse(res *http.Response, retrySec time.Duration, data []byte) error {

	defer res.Body.Close()

	switch res.StatusCode {
	case 200:
		break //all good, all right, but who knows what we could expect from body?

	case 400:
		log.Panic("Google rejected our JSON - this is a critical bug in GCM connector. Fix the bug and retry")

	case 401:
		return GcmAuthError

	case 500:
		log.Println("GCM internal server error, beginning exponential retry")

		if e := gcm.expRetry(res, retrySec, data); e != GcmWontTryAgain {
			return e
		}

		return GcmInternalServerError

	case res.StatusCode >= 501 && res.StatusCode <= 599:
		log.Println("GCM timeout, beginning exponential retry")

		if e := gcm.expRetry(res, retrySec, data); e != GcmWontTryAgain {
			return e
		}

		return GcmTimeoutError

	default:
		return GcmUnknownStatusError
	}

	return gcm.bodyParse(res, retrySec)

}

func (gcm *gcm) regidsPush(regids []string, data Message) error {

	gcmP := &gcmPayload{
		RegIds: regids,
		Data:   data,
	}

	b, e := json.Marshal(gcmP)

	if e != nil {
		return e
	}

	return gcm.request(b, time.Second)

}

func (gcm *gcm) request(data []byte, retryTime time.Duration) error {

	req, e := http.NewRequest("POST", gcmRequestUrl, bytes.NewReader(data))

	if e != nil {
		return e

	}

	req.Header.Add("Authorization", gcm.authKey)

	res, e := gcm.client.Do(req)

	if e != nil {
		return e
	}

	return gcm.evalResponse(res, retryTime, data)

}
