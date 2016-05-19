// Package dashboard implements a web dashboard for monitoring.
package dashboard // import "hkjn.me/dashboard"

import (
	"errors"
	"sync"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"hkjn.me/config"
	"hkjn.me/googleauth"
	"hkjn.me/probes"
)

var (
	emailTemplate = `{{define "email"}}
The probe <a href="http://j.mp/hkjndash#{{.Name}}">{{.Name}}</a> failed enough that this alert fired, as the arbitrary metric of 'badness' is {{.Badness}}, which we can all agree is a big number.<br/>
The description of the probe is: &ldquo;{{.Desc}}&rdquo;<br/>
Failure details follow:<br/>
{{range $r := .Records.RecentFailures}}
  <h2>{{$r.Timestamp}} ({{$r.Ago}})</h2>
  <p>{{$r.Result.Info}}</p>
{{end}}
{{end}}`
	probecfg = struct {
		WebProbes []struct {
			Target, Want, Name string
			WantStatus         int
		}
		DnsProbe struct {
			Target  string
			Records struct {
				Cname string
				A     []string
				Mx    []struct {
					Host string
					Pref uint16
				}
				Ns  []string
				Txt []string
			}
		}
	}{}
	loadConfigOnce = sync.Once{}
)

func setProbesCfg(sgUser, sgPassword, emailSender, emailRecipient, emailTemplate string) error {
	glog.V(1).Infof("Our sendgrid.com user is %q\n", sgUser)
	if sgUser == "" {
		return errors.New("no sendgrid user")
	}
	probes.Config.Sendgrid.User = sgUser
	if sgPassword == "" {
		return errors.New("no sendgrid password")
	}
	probes.Config.Sendgrid.Password = sgPassword

	if emailSender == "" {
		return errors.New("no email sender")
	}
	glog.V(1).Infof("Sending any alert emails from %q to %q\n", emailSender, emailRecipient)
	probes.Config.Alert.Sender = emailSender
	if emailRecipient == "" {
		return errors.New("no email recipient")
	}
	probes.Config.Alert.Recipient = emailRecipient
	if emailTemplate == "" {
		return errors.New("no email template")
	}
	probes.Config.Template = emailTemplate
	return nil
}

func setGoogleAuthCfg(serviceID, serviceSecret string, allowedIDs []string) error {
	if serviceID == "" {
		return errors.New("no service ID")
	}
	glog.V(1).Infof("Our Google service ID is %q\n", serviceID)
	if serviceSecret == "" {
		return errors.New("no service secret")
	}
	googleauth.SetCredentials(serviceID, serviceSecret)
	if len(allowedIDs) == 0 {
		return errors.New("no allowed IDs")
	}
	glog.V(1).Infof("These Google+ IDs are allowed access: %q\n", allowedIDs)
	googleauth.SetGatingFunc(func(id string) bool {
		for _, aid := range allowedIDs {
			if id == aid {
				return true
			}
		}
		return false
	})
	return nil
}

// Start returns the HTTP routes for the dashboard.
//
// If config.yaml is missing, the dashboard will remain inactive, but
// we'll keep retrying to load it on every page request.
func Start(
	googleServiceID,
	googleSecret,
	sgUser,
	sgPassword,
	emailSender,
	emailRecipient string,
	allowedGoogleIDs []string) *mux.Router {
	config.MustLoad(&probecfg, config.Name("probes.yaml"))

	ps := getProbes()
	glog.Infof("Starting %d probes..\n", len(ps))
	for _, p := range ps {
		go p.Run()
	}

	if err := setProbesCfg(sgUser, sgPassword, emailSender, emailRecipient, emailTemplate); err != nil {
		glog.Fatalf("FATAL: Couldn't set probes config: %v\n", err)
	}
	if err := setGoogleAuthCfg(googleServiceID, googleSecret, allowedGoogleIDs); err != nil {
		glog.Fatalf("FATAL: Couldn't set googleauth config: %v\n", err)
	}
	return newRouter()
}
