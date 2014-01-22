package server

import (
	"github.com/mcilloni/pushd/backend"
	"log"
	"net"
)

const (
	BuffSize = 10
)

func Serve(config *config) (e error) {

	log.Printf("Starting server...")

	var (
		failure  = make(chan bool)
		forward  = make(chan command, 10)
		incoming = make(chan net.Conn, 10)
		srv      net.Listener
	)

	if e = backend.InitGcm(config.Gcm); e != nil {
		return
	}

	if e = backend.ConnectDb(config.Postgres); e != nil {
		return
	}

	defer backend.CloseDb()

	if config.Listen.TcpInfo != "" {
		srv, e = net.Listen("tcp", config.Listen.TcpInfo)
	} else {
		srv, e = net.Listen("unix", config.Listen.Socket)
	}

	if e != nil {
		return e
	}

	defer srv.Close()

	for i := uint8(0); i < config.Dispatchers; i++ {
		go dispatch(incoming, forward)
	}

	go func() {
		for {
			conn, e := srv.Accept()

			if e != nil {
				log.Println("Will stop accepting connections")
			    failure <- true //if the error is real (and not caused by close) this will close the server.
				break
			}

			incoming <- conn
		}
	}()

	select {

	case <-failure:

	case f := <-forward:

		switch f {
		case halt:
			break

		default:
			panic("Dispatcher broken - non-halt command recvd")
		}

	}

	close(incoming)
	close(forward)
	log.Println("Server is halting")

}
