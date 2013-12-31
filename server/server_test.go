package server

import (
	"math/rand"
	"testing"
	"time"
)

/*func TestCreate(t *testing.T) {
    if e :=InitDb("user=pushd dbname=pushd host=/run/postgresql sslmode=disable"); e != nil {
        t.Error(e.Error())
    }
}*/

const fakeRegId = "ASH3R2R2FOIC2CRMOMXWHRXR24C22"

var username int64
var dbInst *db

func TestConnect(t *testing.T) {

	rand.Seed(time.Now().Unix())

	var e error

	dbInst, e = dialDb("user=pushd dbname=pushd host=/run/postgresql sslmode=disable")

	if e != nil {
		t.Error(e)
		return
	}

}

func TestAddUser(t *testing.T) {

	username = rand.Int63()

	t.Logf("Adding user %d", username)

	e := dbInst.userAdd(username)

	if e != nil {
		t.Error(e)
		return
	}

	t.Logf("Checking if database contains %d", username)

	users, e := dbInst.users()

	if e != nil {
		t.Error(e)
		return
	}

	ok := false

	for el := users.Front(); el != nil; el = el.Next() {

        if el.Value.(int64) == username {
			ok = true
		}

	}

	if !ok {
		t.Error("Userlist not containing newly subscribed user")
		return
	}

}

func TestAddGcmRegId(t *testing.T) {
	t.Logf("Adding fake RegID to user %d in db", username)

	if e := dbInst.addGcmRegistrationId(username, fakeRegId); e != nil {
		t.Error(e)
		return
	}

}

func TestDeleteUser(t *testing.T) {

	t.Logf("Deleting user %d", username)

	e := dbInst.userDel(username)

	if e != nil {
		t.Error(e)
		return
	}

	t.Logf("Checking if database still contains %d", username)

	users, e := dbInst.users()

	if e != nil {
		t.Error(e)
		return
	}

	ok := false

    for el := users.Front(); el != nil; el = el.Next() {

        if el.Value.(int64) == username {
			ok = true
	    }

	}

	if ok {
		t.Error("Userlist still contains a deleted user")
		return
	}

}

func TestGcmCascade(t *testing.T) {

    t.Log("Checking if db delete cascade deleted regid on userdel")

    regids,e := dbInst.getRegistrationIdsForId(username)

    if e != nil {
        t.Error(e)
        return
    }

    ok := false

    for el := regids.Front(); el != nil; el = el.Next() {

        if el.Value.(string) == fakeRegId {
            ok = true
        }

    }

    if ok {
        t.Error("Cascade failed")
        return
    }

}

func TestCloseDb(t *testing.T) {

	if e := dbInst.close(); e != nil {
		t.Error(e)
	}

}


