// gomon is a web tool that handles monitoring and alerting.
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/golang/glog"

	"hkjn.me/dashboard"
)

var bindAddress = ":8080"

func main() {
	flag.Parse()
	glog.Infoln("gomon initializing..")
	glog.Infof("Listening on %s..\n", bindAddress)
	log.Fatal(http.ListenAndServe(bindAddress, dashboard.Start()))
}
