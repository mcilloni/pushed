/*  Pushed - a daemon for parallel handling of push operations to mobile devices
 *  Copyright (C) 2014  Marco Cilloni <marco.cilloni@yahoo.com>
 *
 *  This Source Code Form is subject to the terms of the Mozilla Public
 *  License, v. 2.0. If a copy of the MPL was not distributed with this
 *  file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *  Exhibit B is not attached; this software is compatible with the
 *  licenses expressed under Section 1.12 of the MPL v2.
 */

package server

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/mcilloni/pushed/backend"
)

const (
	DefaultDispatchers uint8 = 10
)

var (
	ErrConnClosed        = errors.New("Connection to client closed before request")
	routines      uint64 = 0
)

func dispatch(incoming chan net.Conn, forward chan<- command, finished chan<- bool) {

	routineN := atomic.AddUint64(&routines, 1)

	var (
		request, data []byte
		e             error
		read          *bufio.Reader
	)

	logerr := func(e error) {
		if e != io.EOF {
			log.Printf("Error in routine %d: %s", routineN, e.Error())
		}
	}

	for in := range incoming {

		read = bufio.NewReader(in)

		request, e = read.ReadBytes('\n')

		if e != nil {
			logerr(e)
			continue
		}

		data, e = read.ReadBytes('\n')

		if e != nil {
			logerr(e)
			in.Close()
			continue
		}

		op, resp := parseRequest(request, data)

		e = resp.dump(in)

		if e != nil {
			logerr(e)
			in.Close()
			continue
		}

		if resp.Status == accepted {

			if e = execOp(op, forward); e != nil {
				logerr(e)
			}
		}

		if resp.Status == rejected || op.Command != halt {
			incoming <- in //send connection back for further operations
		} else {
			in.Close()
		}

	}

	finished <- true
}

func execOp(op *operation, forward chan<- command) (e error) {
	switch op.Command {

	case halt:
		time.Sleep(op.Parameters[0].(time.Duration))
		forward <- op.Command
		break

	case adduser:
		e = backend.AddUser(op.Parameters[0].(int64))
		break

	case deluser:
		e = backend.DelUser(op.Parameters[0].(int64))
		break

	case subscribe:
		conn := op.Parameters[1].(backend.Connector)
		e = conn.Register(op.Parameters[0].(int64), op.Parameters[2].(string))
		break

	case unsubscribe:
		conn := op.Parameters[1].(backend.Connector)
		e = conn.Unregister(op.Parameters[2].(string))
		break

	case push:
		failed, failures := backend.PushAll(op.Parameters[0].(int64), op.Parameters[1].(backend.Message))

		if failed {
			buffer := bytes.NewBufferString("Errors from connectors - ")
			for key, value := range failures {
				buffer.WriteString(key)
				buffer.WriteString(": '")
				buffer.WriteString(value.Error())
				buffer.WriteString("' ")
			}

			e = errors.New(buffer.String())

		} else {
			e = nil
		}

		break

	}

	return
}
