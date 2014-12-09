// Package dashboard implements a web dashboard for monitoring.
package dashboard // import "hkjn.me/dashboard"

import (
  
        	"hkjn.me/config"
	"hkjn.me/googleauth"
	"hkjn.me/probes"
)

var (
	emailTemplate = `{{define "email"}}
The probe <a href="http://fixme.com/#{{.Name}}">{{.Name}}</a> failed enough that this alert fired, as the arbitrary metric of 'badness' is {{.Badness}}, which we can all agree is a big number.<br/>
The description of the probe is: &ldquo;{{.Desc}}&rdquo;<br/>
Failure details follow:<br/>
{{range $r := .Records.RecentFailures}}
  <h2>{{$r.Timestamp}} ({{$r.Ago}})</h2>
  <p>{{$r.Details}}</p>
{{end}}
{{end}}`
	// Structure of config.yaml.
	cfg = struct {
		Whitelist []string

		Sendgrid  struct {
User, Password string
		}
		Alerts struct {
			Sender, Recipient string
		}
		Service struct {
			Id, Secret string
		}
	}{}
	probecfg = struct {
		WebProbes []struct {
			Target, Want, Name string
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
)

func init() {
	config.MustLoad(&cfg, config.Name("config.yaml"))
	config.MustLoad(&probecfg, config.Name("probes.yaml"))

	googleauth.SetCredentials(cfg.Service.Id, cfg.Service.Secret)
	googleauth.SetGatingFunc(func(gplusId string) bool {
		for _, id := range cfg.Whitelist {
			if gplusId == id {
				return true
			}
		}
		return false
	})
	probes.Config.Sendgrid = cfg.Sendgrid
probes.Config.Template = emailTemplate
	probes.Config.Alert.Sender = cfg.Alerts.Sender
	probes.Config.Alert.Recipient = cfg.Alerts.Recipient
}
