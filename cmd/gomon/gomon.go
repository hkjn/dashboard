// gomon is a web tool that handles monitoring and alerting.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"hkjn.me/dashboard"
)

var bindAddress = ":8080"

func main() {
	flag.Parse()
	fmt.Printf("gomon initializing, listening on %s..\n", bindAddress)

	allowedGoogleIDs := strings.Split(os.Getenv("ALLOWED_GOOGLE_IDS"), ",")
	log.Fatal(http.ListenAndServe(bindAddress, dashboard.Start(
		os.Getenv("GOOGLE_SERVICE_ID"),
		os.Getenv("GOOGLE_SECRET"),
		os.Getenv("SENDGRID_USER"),
		os.Getenv("SENDGRID_PASSWORD"),
		os.Getenv("EMAIL_SENDER"),
		os.Getenv("EMAIL_RECIPIENT"),
		allowedGoogleIDs,
	)))
}
