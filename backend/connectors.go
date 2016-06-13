/*  Pushed - a daemon for parallel handling of push operations to mobile devices
 *  Copyright (C) 2014  Marco Cilloni <marco.cilloni@yahoo.com>
 *
 *  This Source Code Form is subject to the terms of the Mozilla Public
 *  License, v. 2.0. If a copy of the MPL was not distributed with this
 *  file, You can obtain one at http://mozilla.org/MPL/2.0/.
 *  Exhibit B is not attached; this software is compatible with the
 *  licenses expressed under Section 1.12 of the MPL v2.
 */

package backend

import (
	"errors"
	"strings"
	"sync"
)

const (
	connectorSliceSize = 1
)

var (
	ErrNotRegistered = errors.New("Not registered to this connector")
	Gcm              Connector
	connectors       map[string]Connector
	gcmInitOnce      sync.Once
)

type Connector interface {
	Exists(deviceTargetId string) (bool, error)
	Push(user int64, message Message) error
	Register(user int64, deviceTargetId string) error
	Subscribed(user int64) (bool, error)
	Unregister(deviceTargetId string) error
}

func init() {
	connectors = make(map[string]Connector)
}

func ExistsConnector(name string) bool {
	_, ok := connectors[strings.ToLower(name)]
	return ok
}

func GetConnector(name string) Connector {
	return connectors[strings.ToLower(name)]
}

func InitGcm(config *GcmConfig) error {
	gcmInitOnce.Do(func() {
		Gcm = newGcm(config)
		connectors["gcm"] = Gcm
	})

	return nil
}

func PushAll(user int64, message Message) (failures bool, errors map[string]error) {

	errors = make(map[string]error)

	errChan := make(chan error)

	for _, connector := range connectors {
		go func() {
			errChan <- connector.Push(user, message)
		}()
	}

	for name, _ := range connectors {
		e := <-errChan

		if e != nil && e != ErrNotRegistered {
			errors[name] = e
			failures = true
		}
	}

	return
}
