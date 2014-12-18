// Package dashboard implements a web dashboard for monitoring.
package dashboard // import "hkjn.me/dashboard"

import (
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
  <p>{{$r.Details}}</p>
{{end}}
{{end}}`
	cfg      = configT{}
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
	loadConfigOnce = sync.Once{}
)

// Structure of config.yaml.
type configT struct {
	loaded    bool
	Live      bool
	Version   string
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
}

func (c *configT) Loaded() bool {
	return c.loaded
}

func (c *configT) Load() error {
	err := config.Load(&cfg, config.Name("config.yaml"))
	if err != nil {
		return err
	}
	glog.Infoln("successfully loaded config")
	c.loaded = true

	glog.Infoln("Starting probes..")
	for _, p := range getProbes() {
		go p.Run()
	}

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
	return nil
}

func Live() bool { return cfg.Live }

// Start returns the HTTP routes for the dashboard.
//
// If config.yaml is missing, the dashboard will remain inactive, but
// we'll keep retrying to load it on every page request.
func Start() *mux.Router {
	config.MustLoad(&probecfg, config.Name("probes.yaml"))

	err := cfg.Load()
	if err != nil {
		glog.Warningf("couldn't load config.yaml: %v\n", err)
	}
	return newRouter()
}
