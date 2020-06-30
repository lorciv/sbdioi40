package main

import (
	"flag"
	"fmt"
	"log"
	"os"

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
var clean = flag.Bool("clean", true, "Remove temporary files from local storage after migration")

func main() {
	flag.Parse()

	appname := flag.Arg(0)
	if appname == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] appname\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Migrate a SBDIOI40 application between the two given platforms.")
		fmt.Fprintln(os.Stderr, "options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	srcPlat, err := sbdioi40.Connect(*srcAddr, *username, *password)
	if err != nil {
		log.Fatal(err)
	}
	dstPlat, err := sbdioi40.Connect(*dstAddr, *username, *password)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("connected successfully to both platforms")

	snap, err := srcPlat.Snapshot(appname)
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
