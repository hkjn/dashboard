package dashboard

import (
	"flag"
	"net/http"
	"sync"

	"github.com/golang/glog"
	"hkjn.me/dashboard/googleauth"
)

var (
	authDisabled = flag.Bool("no_auth", false, "disables authentication (use for testing only)")
	parseFlags   sync.Once
)

// requireLogin returns a handler that enforces Google+ login.
func (fn handlerFunc) requireLogin() handlerFunc {
	parseFlags.Do(func() {
		if !flag.Parsed() {
			flag.Parse()
		}
	})
	return func(w http.ResponseWriter, r *http.Request) {
		loggedIn := false
		var err error
		if *authDisabled {
			glog.Infof("-disabled_auth is set, not checking credentials\n")
			loggedIn = true
		} else {
			loggedIn, err = googleauth.IsLoggedIn(r)
			if err != nil {
				glog.Errorf("failed to get login info: %v\n", err)
				serveISE(w)
				return
			}
		}
		if loggedIn {
			glog.V(1).Infof("user is logged in, onward to original render func\n")
			fn(w, r)
			return
		}
		glog.V(1).Infof("not logged in, fetching state token\n")
		li, err := googleauth.LogIn(w, r)
		if err != nil {
			glog.Errorf("failed to get login info: %v\n", err)
			serveISE(w)
			return
		}
		err = loginTmpl.ExecuteTemplate(w, "login", li)
		if err != nil {
			serveISE(w)
			return
		}
	}
}

// connect exchanges the one-time authorization code for a token and stores the
// token in the session
func connect(w http.ResponseWriter, r *http.Request) {
	err := googleauth.Connect(w, r)
	if err != nil {
		if googleauth.IsAccessDenied(err) {
			http.Error(w, "Access denied.", http.StatusUnauthorized)
		} else {
			glog.Errorf("error connecting to googleauth: %v\n", err)
			serveISE(w)
		}
		return
	}
	glog.V(1).Infof("current user is connected, redirecting to %q\n", r.Referer())
	http.Redirect(w, r, r.Referer(), http.StatusSeeOther)
}
