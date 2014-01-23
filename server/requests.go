package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mcilloni/pushed/backend"
	"io"
	"strconv"
	"time"
)

type command string
type Status string

const (
	adduser     command = "ADDUSER"
	deluser     command = "DELUSER"
	halt        command = "HALT"
	push        command = "PUSH"
	subscribe   command = "SUBSCRIBE"
	unsubscribe command = "UNSUBSCRIBE"

	accepted Status = "ACCEPTED"
	rejected Status = "REJECTED"
)

type operation struct {
	Command    command
	Parameters []interface{}
}

type response struct {
	Status  Status
	Message string
}

func (resp *response) Dump(w io.Writer) (e error) {

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

func ParseRequest(head, data []byte) (op *operation, resp *response) {

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

			op.Parameters[0] = time.Duration(val)

			break

		default:
			return failure("Too many arguments for %s : %d", fields[0], fieldsLen)
		}

		return

	case adduser, deluser:

		if fieldsLen != 2 {
			return failure("Wrong number of arguments for %s: %d", fields[0], fieldsLen)
		}

		val, e := strconv.ParseInt(string(fields[1]), 10, 64)

		if e != nil {
			return failure("Cannot parse %s as an integer", fields[1])
		}

		op.Parameters = []interface{}{val}

		break

	case subscribe, unsubscribe:

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

		if len(param2) != 2 {
			return failure("Malformed CONNECTOR:ID string")
		}

		conn := backend.GetConnector(string(param2[0]))

		if conn != nil {
			return failure("Connector %s does not exist", param2[0])
		}

		op.Parameters[1], op.Parameters[2] = conn, string(param2[1])

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

func failure(format string, args ...interface{}) (*operation, *response) {
	return nil, newResponse(rejected, format, args...)
}

func newResponse(status Status, format string, args ...interface{}) *response {
	return &response{Status: status, Message: fmt.Sprintf(format, args...)}
}
