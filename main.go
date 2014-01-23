package main

import (
	"flag"
    "fmt"
	"github.com/mcilloni/pushed/server"
	"log"
	"os"
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
    flag.BoolVar(&initDb, "init", false, "initializes PostgreSQL with pushed tables as by the Postgres parameter in conffile. createdb the db first, and ensure you have permissions for the given user")
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

    e := server.Serve(args[0])

	if e != nil {
		log.Fatal(e)
	}

}
