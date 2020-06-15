package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/lorciv/sbdioi40"
)

func init() {
	log.SetPrefix("sbdioi40: ")
	log.SetFlags(0)
}

var addr = flag.String("addr", "", "Address to OpenStack authentication endpoint")
var username = flag.String("user", "", "Username for authentication")
var password = flag.String("pass", "", "Password for authentication")

func main() {
	flag.Parse()

	plat, err := sbdioi40.Connect(*addr, *username, *password)
	if err != nil {
		log.Fatal(err)
	}

	apps, err := plat.ListApplications()
	if err != nil {
		log.Fatal(err)
	}

	for _, app := range apps {
		fmt.Println(app)
	}
}