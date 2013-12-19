package server

import (
	"errors"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

func TestSubUnsub(t *testing.T) {

	rand.Seed(time.Now().Unix())

	var e error

	defer func() {
		if e != nil {
			t.Error(e.Error())
		}
	}()

	db, e := DialDb("tcp", "localhost:6379")

	if e != nil {
		return
	}

	username := "testuser" + strconv.FormatInt(rand.Int63(), 10)
	connector := "randomconnector"

	e = db.Subscribe(username, connector)

	if e != nil {
		return
	}

	t.Logf("Checking if database contains %s", username)

	users, e := db.Users()

	if e != nil {
		return
	}

	ok := false

	for _, user := range users {
		if user == username {
			ok = true
		}
	}

	if !ok {
		e = errors.New("Userlist not containing newly subscribed user")
		return
	}

	e = db.Unsubscribe(username, connector)

	if e != nil {
		return
	}

	t.Logf("Checking if database still contains %s", username)

	users, e = db.Users()

	if e != nil {
		return
	}

	ok = false

	for _, user := range users {
		if user == username {
			ok = true
		}
	}

	if ok {
		e = errors.New("Userlist still contains a deleted user")
		return
	}

}
