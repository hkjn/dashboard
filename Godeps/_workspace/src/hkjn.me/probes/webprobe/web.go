// Package webprobe implements a HTTP probe.
package webprobe

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"hkjn.me/prober"
	"hkjn.me/probes"
)

var (
	MaxResponse int64 = 1000000 // largest response size accepted
	defaultName       = "WebProber"
)

// WebProber probes a target's HTTP response.
type WebProber struct {
	Target         string // URL to probe
	Method         string // GET, POST, PUT, etc.
	Name           string // name of the prober
	Body           io.Reader
	wantCode       int
	wantInResponse string
}

// Name sets the name for the prober.
func Name(name string) func(*WebProber) {
	return func(p *WebProber) {
		p.Name = fmt.Sprintf("%s_%s", defaultName, name)
	}
}

// Body sets the HTTP request body for the prober.
func Body(body io.Reader) func(*WebProber) {
	return func(p *WebProber) {
		p.Body = body
	}
}

// InResponse applies the option that the prober wants given string in the HTTP response.
func InResponse(str string) func(*WebProber) {
	return func(p *WebProber) {
		p.wantInResponse = str
	}
}

// New returns a new instance of the web probe with specified options.
func New(target, method string, code int, options ...func(*WebProber)) *prober.Probe {
	name := defaultName
	p := &WebProber{Target: target, Name: name, Method: method, wantCode: code}
	for _, opt := range options {
		opt(p)
	}
	return prober.NewProbe(p, p.Name, fmt.Sprintf("Probes HTTP response of %s", target))
}

// Probe verifies that the target's HTTP response is as expected.
func (p WebProber) Probe() error {
	req, err := http.NewRequest(p.Method, p.Target, p.Body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	c := http.Client{}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-success HTTP response: %q", resp.Status)
	}
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, MaxResponse))
	if err != nil {
		return fmt.Errorf("failed to read HTTP response: %v", err)
	}
	sb := string(body)
	if !strings.Contains(sb, p.wantInResponse) {
		return fmt.Errorf("response doesn't contain %q: \n%v\n", p.wantInResponse, sb)
	}
	return nil
}

// Alert sends an alert notification via email.
func (p *WebProber) Alert(name, desc string, badness int, records prober.Records) error {
	return probes.SendAlertEmail(name, desc, badness, records)
}
