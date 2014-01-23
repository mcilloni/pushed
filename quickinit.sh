#!/usr/bin/env

#Just a quick and dirty script for user creation. 
#It creates a pushed user, owner of db pushed.
#Remember to set correctly your config and then launch pushed -initdb.

set +e

createuser pushed
createdb -O pushed pushed "Pushed database"
