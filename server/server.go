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
	"log"
	"net"

	"github.com/mcilloni/pushed/backend"
)

const (
	BuffSize = 10
)

func InitDatabase(configPath string) (e error) {

	conf, e := parse(configPath)
	if e != nil {
		return
	}

	return backend.InitDb(conf.Postgres)

}

func Serve(configPath string, stop <-chan bool) (e error) {

	conf, e := parse(configPath)
	if e != nil {
		return
	}

	return serveConfig(conf, stop)

}

func serveConfig(config *config, stop <-chan bool) (e error) {

	log.Printf("Starting server...")

	var (
		failure  = make(chan bool)
		forward  = make(chan command, 10)
		incoming = make(chan net.Conn, 10)
		srv      net.Listener
		wait     = make(chan bool)
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
		go dispatch(incoming, forward, wait)
	}

	go func() {

		log.Println("Server is initialized, accepting connections")

		for {
			conn, e := srv.Accept()

			if e != nil {
				log.Println("Terminating operations")
				failure <- true //if the error is real (and not caused by Close) this will close the server.
				break
			}

			incoming <- conn
		}
	}()

	select {

	case <-failure:
		break
	case f := <-forward:

		switch f {
		case halt:
			break

		default:
			panic("Dispatcher broken - non-halt command recvd")
		}
	case <-stop:
		break

	}

	close(incoming)
	close(forward)
	log.Println("Server is halting")

	for i := uint8(0); i < config.Dispatchers; i++ {
		<-wait //wait for routines to finish
	}

	return
}
