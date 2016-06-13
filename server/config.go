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
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/mcilloni/pushed/backend"
)

type connParams struct {
	TcpInfo string
	Socket  string
}

type config struct {
	Listen      *connParams
	Postgres    string
	Gcm         *backend.GcmConfig
	Dispatchers uint8
}

func parse(confPath string) (conf *config, e error) {

	log.Println("Parsing JSON config file " + confPath)

	fileContents, e := ioutil.ReadFile(confPath)

	if e != nil {
		return
	}

	var values config

	e = json.Unmarshal(fileContents, &values)

	if e != nil {
		return
	}

	if (values.Listen.TcpInfo != "") == (values.Listen.Socket != "") {
		return nil, errors.New("both (neither) port and (nor) socket are specified on configuration file " + confPath)
	}

	if values.Listen.Socket != "" {

		socket := values.Listen.Socket

		if !path.IsAbs(socket) {
			return nil, errors.New("given path " + socket + "is not absolute")
		}

		if _, err := os.Stat(socket); err == nil {
			return nil, errors.New("cannot create a socket on already existing file " + socket)
		}

	}

	if values.Postgres == "" {
		return nil, errors.New("No postgres connection string in " + confPath)
	}

	if values.Gcm != nil {
		if values.Gcm.ApiKey == "" {
			return nil, errors.New("Gcm config object set but no ApiKey field set")
		}

		if values.Gcm.MaxRetryTime != 0 {
			values.Gcm.MaxRetryTime *= time.Second //I don't expect people to input nanoseconds ;)
		}
	}

	if values.Dispatchers == 0 {
		values.Dispatchers = DefaultDispatchers
	}

	return &values, nil

}
