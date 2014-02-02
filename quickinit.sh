#!/usr/bin/env bash

#Just a quick and dirty script for user creation. 
#It creates a pushed user, owner of db pushed.
#Remember to set correctly your config and then launch pushed -initdb.

set +e

if [ $# -lt 1 ]; then
    echo "Usage: $0 postgres_user"
    exit -1
fi

echo "Adding pushed PostgreSQL user"
createuser -U $1 pushed

echo "Adding pushed PostgreSQL database"
createdb -U $1 -O pushed pushed "Pushed database"
