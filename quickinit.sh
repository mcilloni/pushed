#!/usr/bin/env bash

#Just a quick and dirty script for user creation. 
#It creates a pushed user, owner of db pushed.
#Remember to set correctly your config and then launch pushed -initdb.

set +e

echo "Adding pushed PostgreSQL user"
createuser pushed

echo "Adding pushed PostgreSQL database"
createdb -O pushed pushed "Pushed database"
