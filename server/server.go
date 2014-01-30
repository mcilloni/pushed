/*  Pushed - a daemon for parallel handling of push operations to mobile devices
 *  Copyright (C) 2014  Marco Cilloni <marco.cilloni@yahoo.com>
 *
 *  This program is free software: you can redistribute it and/or modify
 *  it under the terms of the GNU Affero General Public License as
 *  published by the Free Software Foundation, either version 3 of the
 *  License, or (at your option) any later version.
 *
 *  This program is distributed in the hope that it will be useful,
 *  but WITHOUT ANY WARRANTY; without even the implied warranty of
 *  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *  GNU Affero General Public License for more details.
 *
 *  You should have received a copy of the GNU Affero General Public License
 *  along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package server

import (
	"github.com/mcilloni/pushed/backend"
	"log"
	"net"
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
