// gomon is a web tool that handles monitoring and alerting.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/golang/glog"

	"hkjn.me/dashboard"
)

func createProbes() {
	glog.Infof("Starting probes..\n")
	for _, p := range dashboard.GetProbes() {
		go p.Run()
	}
}

func main() {
	if !flag.Parsed() {
		flag.Parse()
	}

	router := dashboard.NewRouter()

	createProbes()

	log.Fatal(http.ListenAndServe(":8080", router))
}
