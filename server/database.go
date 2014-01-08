package server

import (
	"container/list"
	"database/sql"
	"errors"
	_ "github.com/lib/pq"
	"log"
	"sync"
)

type db struct {
	conn                                                        *sql.DB
	userAddStmt, userDelStmt, gcmRegAdd, gcmRegDel, gcmRegFetch *sql.Stmt
}

var (
	ErrUserNotExisting = errors.New("User is not existant")
	ErrUserExists      = errors.New("User already exists")
)

func dialDb(connstr string) (*db, error) {
	log.Println("Connecting to postgresql...")
	conn, e := sql.Open("postgres", connstr)

	if e != nil {
		return nil, e
	}

	if e = conn.Ping(); e != nil {
		return nil, e
	}

	dbInst := new(db)

	dbInst.conn = conn

	e = gcmInitStmt(dbInst)

	if e != nil {
		return nil, e
	}

	dbInst.userAddStmt, e = conn.Prepare("INSERT INTO USERS VALUES ($1)")

	if e != nil {
		return nil, e
	}

	dbInst.userDelStmt, e = conn.Prepare("DELETE FROM USERS WHERE ID = $1")

	if e != nil {
		return nil, e
	}

	return dbInst, nil
}

func (db *db) close() (e error) {

	if e = gcmCloseStmt(db); e != nil {
		return
	}

	if e = db.userAddStmt.Close(); e != nil {
		return
	}

	if e = db.userDelStmt.Close(); e != nil {
		return
	}

	return db.conn.Close()

}

func (db *db) probe() error {
	return db.conn.Ping()
}

func (db *db) users() (*list.List, error) {

	people := list.New()

	rows, e := db.conn.Query("SELECT ID FROM USERS")

	if e != nil {
		return nil, e
	}

	var id int64

	for rows.Next() {
		if e = rows.Scan(&id); e != nil {
			return nil, e
		}

		people.PushBack(id)
	}

	if e = rows.Err(); e != nil {
		return nil, e
	}

	return people, nil

}

func (db *db) userAdd(id int64) error {
	log.Printf("Adding user %d...", id)

	_, e := db.userAddStmt.Exec(id)

	return e
}

func (db *db) userDel(id int64) error {
	log.Printf("Deleting user %d...", id)

	_, e := db.userDelStmt.Exec(id)

	return e

}

func InitDb(connstr string) error {

	log.Println("Connecting to postgresql...")
	conn, e := sql.Open("postgres", connstr)

	if e != nil {
		return e
	}

	if e = conn.Ping(); e != nil {
		return e
	}

	dbInst := new(db)

	dbInst.conn = conn

	if e != nil {
		return e
	}

	log.Println("Connected.\nCreating table USERS...")

	_, e = dbInst.conn.Exec("CREATE TABLE USERS (ID BIGINT PRIMARY KEY CHECK (ID > -1))")

	if e != nil {
		return e
	}

	log.Println("Done.\nCreating table GCM...")

	if e = dbInst.gcmInitTable(); e != nil {
		return e
	}

	log.Println("Done.")

	return nil

}
