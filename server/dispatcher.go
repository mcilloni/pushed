package server

import (
	"bufio"
	"errors"
	"github.com/mcilloni/pushed/backend"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
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

	log.Printf("Dispatcher routine %d started", routineN)

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

		e = resp.Dump(in)

		if e != nil {
			logerr(e)
			in.Close()
			continue
		}

		if resp.Status == accepted {

			if e = execOp(op, forward); e != nil {
				logerr(e)
				in.Close()
				continue
			}
		}

        if op.Command != halt {
            incoming <- in //send connection back for further operations
        } else {
            in.Close()
        }

	}

	log.Printf("Dispatcher routine %d exited", routineN)

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
		conn := op.Parameters[1].(backend.Connector)
		e = conn.Push(op.Parameters[0].(int64), op.Parameters[0].(backend.Message))
		break

	}

	return
}
