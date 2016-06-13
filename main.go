/*  Pushed - a daemon for parallel handling of push operations to mobile devices
 *  Copyright (C) 2014  Marco Cilloni <marco.cilloni@yahoo.com>
 *
 *  This Source Code Form is subject to the terms of the Mozilla Public
 *  License, v. 2.0. If a copy of the MPL was not distributed with this
 *  file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *  Exhibit B is not attached; this software is compatible with the
 *  licenses expressed under Section 1.12 of the MPL v2.
 */

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"

	"github.com/mcilloni/pushed/server"
)

var (
	confPath string
	help     bool
	initDb   bool
	logPath  string
)

func init() {
	flag.BoolVar(&help, "help", false, "prints this help")
	flag.BoolVar(&help, "h", false, "shorthand for -help")
	flag.BoolVar(&initDb, "initdb", false, "initializes PostgreSQL with pushed tables as by the Postgres parameter in conffile. createdb the db first, and ensure you have permissions for the given user")
	flag.StringVar(&logPath, "logfile", "", "sets the path of the pushed log file. If not set, it will default to stdout")
	flag.StringVar(&logPath, "l", "", "shorthand for -logfile")
}

func printHelp() {
	fmt.Println("usage: pushed [params] conffile.json")
	flag.PrintDefaults()
}

func main() {

	var logFile *os.File

	//If panic during execution, recover, log and exit
	defer func() {
		if r := recover(); r != nil {
			log.Println(string(debug.Stack()))
		}

		if logFile != nil {
			logFile.Close()
		}
	}()

	flag.Parse()

	if help {
		printHelp()
		return
	}

	if logPath != "" {
		logFile, e := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)

		if e != nil {
			fmt.Printf("Cannot open %s: %s\n", logPath, e.Error())
			return
		}

		log.SetOutput(logFile)
	}

	args := flag.Args()

	if len(args) != 1 {
		fmt.Println("Wrong number of arguments")
		printHelp()
		return
	}

	var e error

	if initDb {
		e = server.InitDatabase(args[0])
	} else {

		interr := make(chan os.Signal, 1)
		stop := make(chan bool, 1)
		signal.Notify(interr, os.Interrupt)

		go func() {
			<-interr
			stop <- true
		}()

		e = server.Serve(args[0], stop)
	}

	if e != nil {
		log.Fatal(e)
	}

}
