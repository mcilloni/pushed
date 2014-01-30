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
	Exists(user int64, deviceTargetId string) (bool, error)
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
