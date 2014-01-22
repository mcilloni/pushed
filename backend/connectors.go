package backend

import (
	"strings"
	"sync"
)

const (
	connectorSliceSize = 1
)

var (
	Gcm         Connector
	connectors  map[string]Connector
	gcmInitOnce sync.Once
)

type Connector interface {
	Push(user int64, message Message) error
	Register(user int64, deviceTargetId string) error
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
		errors[name] = <-errChan
		if errors[name] != nil {
			failures = true
		}
	}

	return
}
