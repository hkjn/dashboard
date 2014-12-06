// gomon is a web tool that handles monitoring and alerting.
package main

import (
	"log"
	"net/http"

	"hkjn.me/dashboard"
)

func main() {
	for _, p := range dashboard.GetProbes() {
		go p.Run()
	}
	log.Fatal(http.ListenAndServe(":8080", dashboard.NewRouter()))
}
