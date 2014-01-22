package main

import (
	"flag"
	"github.com/mcilloni/pushd/server"
	"log"
	"os"
)

var (
	confPath string
	help     bool
	logPath  string
)

func init() {
	flag.StringVar(&confPath, "confpath", "", "Sets the path of a JSON pushd config file. Mandatory, can be shortened to -c")
	flag.StringVar(&confPath, "c", "", "Shorthand for confpath")
	flag.BoolVar(&help, "help", false, "Prints this help")
	flag.BoolVar(&help, "h", false, "Shorthand for -help")
	flag.StringVar(&logPath, "logfile", "", "Sets the path of the pushd log file. If not set, it will default to stdout")
	flag.StringVar(&logPath, "l", "", "Shorthand for -logfile")
}

func main() {
	flag.Parse()

	if help {
		flag.PrintDefaults()
		return
	}

	if logPath != "" {
		logFile, e := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND, 0666)

		if e != nil {
			log.Fatalf("Cannot open %s: %s", logPath, e.Error())
		}

		defer logFile.Close()

		log.SetOutput(logFile)
	}

	if confPath == "" {
		log.Fatal("No config path given.")
	}

    e := server.Serve(confPath)

	if e != nil {
		log.Fatal(e)
	}

}
