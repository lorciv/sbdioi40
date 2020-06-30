package main

import (
	"flag"
	"log"

	"github.com/lorciv/sbdioi40"
)

func init() {
	log.SetPrefix("sbdioi40: ")
	log.SetFlags(0)
}

var srcAddr = flag.String("src", "", "Address to the source OpenStack platform (authentication endpoint)")
var dstAddr = flag.String("dst", "", "Address to the destination OpenStack platform (authentication endpoint)")
var username = flag.String("user", "", "Username for authentication on both platforms")
var password = flag.String("pass", "", "Password for authentication on both platforms")
var appname = flag.String("app", "", "Name of the application to migrate")
var clean = flag.Bool("clean", true, "Remove temporary files from local storage after migration")

func main() {
	flag.Parse()

	srcPlat, err := sbdioi40.Connect(*srcAddr, *username, *password)
	if err != nil {
		log.Fatal(err)
	}
	dstPlat, err := sbdioi40.Connect(*dstAddr, *username, *password)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("connected successfully to both platforms")

	snap, err := srcPlat.Snapshot(*appname)
	if err != nil {
		log.Fatal(err)
	}
	log.Print(snap)

	if err := dstPlat.Restore(snap); err != nil {
		log.Fatal(err)
	}

	if *clean {
		snap.Remove()
	}

	log.Print("done")
}
