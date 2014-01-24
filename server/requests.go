package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcilloni/pushed/backend"
	"io"
	"log"
	"strconv"
	"time"
)

type command string
type Status string

const (
	adduser     command = "ADDUSER"
	deluser     command = "DELUSER"
	exists      command = "EXISTS"
	halt        command = "HALT"
	push        command = "PUSH"
	subscribe   command = "SUBSCRIBE"
	subscribed  command = "SUBSCRIBED"
	unsubscribe command = "UNSUBSCRIBE"

	accepted Status = "ACCEPTED"
	no       Status = "NO"
	rejected Status = "REJECTED"
	yes      Status = "YES"
)

var (
	noResp  = newResponse(no, "Not existent")
	yesResp = newResponse(yes, "Exists")
)

type operation struct {
	Command    command
	Parameters []interface{}
}

type response struct {
	Status  Status
	Message string
}

func (resp *response) dump(w io.Writer) (e error) {

	buffer := bytes.NewBufferString(string(resp.Status))

	e = buffer.WriteByte(' ')
	if e != nil {
		return
	}

	_, e = buffer.WriteString(resp.Message)
	if e != nil {
		return
	}

	e = buffer.WriteByte('\n')
	if e != nil {
		return
	}

	_, e = buffer.WriteTo(w)

	return

}

func failure(format string, args ...interface{}) (*operation, *response) {
	return nil, newResponse(rejected, format, args...)
}

func newResponse(status Status, format string, args ...interface{}) *response {
	return &response{Status: status, Message: fmt.Sprintf(format, args...)}
}

func parseRequest(head, data []byte) (op *operation, resp *response) {

	fields := bytes.Fields(head)

	fieldsLen := len(fields)

	if fieldsLen == 0 {
		return failure("Header too short")
	}

	op = &operation{Command: command(fields[0])}
	resp = &response{Status: accepted, Message: "Request accepted."}

	switch op.Command {
	case halt:

		op.Parameters = make([]interface{}, 1)

		switch fieldsLen {
		case 1:
			op.Parameters[0] = time.Duration(0)
		case 2:
			val, e := strconv.ParseInt(string(fields[1]), 10, 64)

			if e != nil {
				return failure("Cannot parse %s as an integer", fields[1])
			}

			op.Parameters[0] = time.Duration(val) * time.Second

			break

		default:
			return failure("Too many arguments for %s : %d", fields[0], fieldsLen)
		}

		return

	case adduser, deluser, exists:

		if fieldsLen != 2 {
			return failure("Wrong number of arguments for %s: %d", fields[0], fieldsLen)
		}

		val, e := strconv.ParseInt(string(fields[1]), 10, 64)

		if e != nil {
			return failure("Cannot parse %s as an integer", fields[1])
		}

		op.Parameters = []interface{}{val}

		if op.Command == exists {
			resp, e = synchronousRequest(op)

			if e != nil {
				log.Printf("Error: %s", e.Error())
				return failure("Internal error")
			}

			return

		}

		break

	case subscribe, subscribed, unsubscribe:

		reqCheck := op.Command == subscribed

		op.Parameters = make([]interface{}, 3)

		if fieldsLen != 3 {
			return failure("Wrong number of arguments for %s: %d", fields[0], fieldsLen)
		}

		val, e := strconv.ParseInt(string(fields[1]), 10, 64)

		if e != nil {
			return failure("Cannot parse %s as a signed integer", fields[1])
		}

		op.Parameters[0] = val

		param2 := bytes.SplitN(fields[2], []byte(":"), 2)

		lenParam2 := len(param2)

		if lenParam2 != 2 && !(reqCheck && lenParam2 == 1) {
			return failure("Malformed request string")
		}

		conn := backend.GetConnector(string(param2[0]))

		if conn == nil {
			return failure("Connector %s does not exist", param2[0])
		}

		devId := ""

		if lenParam2 == 2 {
			devId = string(param2[1])
		}

		op.Parameters[1], op.Parameters[2] = conn, devId

		if reqCheck {
			resp, e = synchronousRequest(op)

			if e != nil {
				log.Printf("Error: %s", e.Error())
				return failure("Internal error")
			}

			return
		}

		break

	case push:

		if fieldsLen != 2 {
			return failure("Wrong number of arguments for %s: %d", fields[0], fieldsLen)
		}

		val, e := strconv.ParseInt(string(fields[1]), 10, 64)
		if e != nil {
			return failure("Cannot parse %s as a signed integer", fields[1])
		}

		var validData backend.Message

		e = json.Unmarshal(data, &validData)

		if data != nil && e != nil {
			return failure("Malformed json for PUSH request")
		}

		op.Parameters = []interface{}{val, validData}

		break

	default:
		return failure("Unknown request %s", op.Command)

	}

	return

}

func synchronousRequest(op *operation) (resp *response, e error) {

	var b bool

	switch op.Command {
	case exists:
		b, e = backend.Exists(op.Parameters[0].(int64))

		break

	case subscribed:

		conn := op.Parameters[1].(backend.Connector)

		id := op.Parameters[0].(int64)
		devId := op.Parameters[2].(string)

		if len(devId) > 0 {
			b, e = conn.Exists(id, devId)
		} else {
			b, e = conn.Subscribed(id)
		}

		break

	default:
		panic("Cannot call synchronousRequest for " + string(op.Command))
	}

	if b {
		resp = yesResp
	} else {
		resp = noResp
	}

	return
}
