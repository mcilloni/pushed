package main

import (
        "flag" 
        "github.com/mcilloni/pushd/server"
         "log"
)

func main() {
    server.Parse("/home/marco/config.json")
}
