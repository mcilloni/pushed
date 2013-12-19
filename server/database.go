package server

import (
	"errors"
	"fmt"
	"github.com/fzzy/radix/redis"
	"log"
	"strings"
)

const (
	userSpace = "pushdusers:"
)

type Subscriptions []string

type Db struct {
	client *redis.Client
}

func DialDb(network, addr string) (*Db, error) {
	log.Printf("Connecting to %s via %s", addr, network)
	client, e := redis.Dial(network, addr)

	if e != nil {
		return nil, e
	}

	return &Db{client}, nil
}

func (db *Db) GetSubs(id string) (Subscriptions, error) {
	reply := db.client.Cmd("SMEMBERS", userSpace+id)

	switch reply.Type {
	case redis.ErrorReply:
		return nil, reply.Err
	case redis.NilReply:
		return nil, errors.New(id + " not registered")
	case redis.MultiReply:
		break
	default:
		return nil, errors.New("Fatal error, broken redis connector")
	}

	info, e := reply.List()

	if e != nil {
		return nil, e
	}

	return info, nil

}

func (db *Db) Subscribe(name, connector string) error {

	log.Printf("Subscribing %s to %s", name, connector)

	reply := db.client.Cmd("SADD", userSpace+name, connector)
	switch reply.Type {
	case redis.ErrorReply:
		return reply.Err
	case redis.IntegerReply:
		ok, e := reply.Bool()

		if e != nil {
			return e
		}

		if !ok {
			return fmt.Errorf("Failed to add to set %s connector %s", userSpace+name, connector)
		}
		break
	default:
		return errors.New("Broken connector")
	}

	return nil
}

func (db *Db) Unsubscribe(name, connector string) error {

	log.Printf("Unsubscribing %s from %s", name, connector)

	reply := db.client.Cmd("SREM", userSpace+name, connector)

	switch reply.Type {
	case redis.ErrorReply:
		return reply.Err
	case redis.IntegerReply:
		ok, e := reply.Bool()

		if e != nil {
			return e
		}

		if !ok {
			return fmt.Errorf("%s is not subscribed to %s", userSpace+name, connector)
		}
		break
	default:
		return errors.New("Broken connector")
	}

	return nil

}

func (db *Db) Users() ([]string, error) {
	reply := db.client.Cmd("KEYS", userSpace+"*")

	switch reply.Type {
	case redis.ErrorReply:
		return nil, reply.Err
	case redis.NilReply:
		return nil, errors.New("No user registered")
	case redis.MultiReply:
		break
	default:
		return nil, errors.New("Fatal error, broken redis connector")
	}

	list, e := reply.List()

	if e != nil {
		return nil, e
	}

	var keySplit []string

	for i, key := range list {
		keySplit = strings.Split(key, ":")
		if len(keySplit) != 2 {
			return nil, fmt.Errorf("Malformed username %s in %s", key, userSpace)
		}

		list[i] = keySplit[1]

	}

	return list, nil

}
