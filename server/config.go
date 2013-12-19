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

type ConnParams struct {
	Unix   bool
	Host   uint16
	Socket string
}

type GcmParams struct {
}

type Config struct {
	Connection ConnParams
	Gcm        GcmParams
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

	value, okHost := mapValues["host"]

	if okHost {

		if okSocket {
			return nil, errors.New("both host and socket are specified on configuration file " + confPath)
		}

		if host, ok := value.(uint16); ok {
			connParams = ConnParams{Unix: false, Host: host}
		} else {
			return nil, errors.New("invalid host number " + fmt.Sprintf("%s", value))
		}

	}

	if !(okHost || okSocket) {
		return nil, errors.New("neither host nor socket set in " + confPath)
	}

	return &Config{connParams, GcmParams{}}, e

}
