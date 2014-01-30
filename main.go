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

package main

import (
	"flag"
	"fmt"
	"github.com/mcilloni/pushed/server"
	"log"
	"os"
	"os/signal"
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
	flag.Parse()

	if help {
		printHelp()
		return
	}

	if logPath != "" {
		logFile, e := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0666)

		if e != nil {
			fmt.Printf("Cannot open %s: %s\n", logPath, e.Error())
			return
		}

		defer logFile.Close()

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
