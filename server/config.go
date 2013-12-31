package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
)

type PostgresStr string

type ConnParams struct {
	Unix   bool
	Port   uint16
	Socket string
}

type Config struct {
	BindOpt  ConnParams
	PsqlConn PostgresStr
}

func Parse(confPath string) (config *Config, e error) {

	config = nil

	log.Println("Parsing JSON config file " + confPath)

	fileContents, e := ioutil.ReadFile(confPath)

	if e != nil {
		return
	}

	var values interface{}

	e = json.Unmarshal(fileContents, &values)

	if e != nil {
		return
	}

	mapValues := values.(map[string]interface{})

	var connParams ConnParams

	value, okSocket := mapValues["socket"]

	if okSocket {

		socket, ok := value.(string)

		if !ok {
			return nil, errors.New("socket set but not string")
		}

		if !path.IsAbs(socket) {
			return nil, errors.New("given path " + socket + "is not absolute")
		}

		if _, err := os.Stat(socket); err == nil {
			return nil, errors.New("cannot create a socket on already existing file " + socket)
		}

		connParams = ConnParams{Unix: true, Socket: socket}

	}

	value, okPort := mapValues["port"]

	if okPort {

		if okSocket {
			return nil, errors.New("both port and socket are specified on configuration file " + confPath)
		}

		if port, ok := value.(uint16); ok {
			connParams = ConnParams{Unix: false, Port: port}
		} else {
			return nil, errors.New("invalid port number " + fmt.Sprintf("%s", value))
		}

	}

	if !(okPort || okSocket) {
		return nil, errors.New("neither port nor socket set in " + confPath)
	}

	value, okConnStr := mapValues["postgres"]

	if !okConnStr {
		return nil, errors.New("No postgres connection string in " + confPath)
	}

	connStr, ok := value.(string)

	if !ok {
		return nil, errors.New("Field postgres in " + confPath + " is not a string")
	}

	return &Config{connParams, PostgresStr(connStr)}, nil

}
