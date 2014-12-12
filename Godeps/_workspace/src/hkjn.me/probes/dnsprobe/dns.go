// Package dnsprobe implements a DNS probe.
package dnsprobe // import "hkjn.me/probes/dnsprobe"

import (
	"fmt"
	"net"
	"sort"
	"time"

	"github.com/golang/glog"

	"hkjn.me/prober"
	"hkjn.me/probes"
)

// DnsProber probes a target host's DNS records.
type DnsProber struct {
	Target    string // host to probe
	wantMX    mxRecords
	wantA     []string
	wantNS    nsRecords
	wantCNAME string
	wantTXT   []string
}

// MX applies the option that the prober wants specific MX records.
func MX(mx []*net.MX) func(*DnsProber) {
	return func(p *DnsProber) {
		wantMX := mxRecords(mx)
		sort.Sort(wantMX)
		p.wantMX = wantMX
	}
}

// A applies the option that the prober wants specific A records.
func A(a []string) func(*DnsProber) {
	return func(p *DnsProber) {
		sort.Strings(a)
		p.wantA = a
	}
}

// NS applies the option that the prober wants specific NS records.
func NS(ns []*net.NS) func(*DnsProber) {
	return func(p *DnsProber) {
		nsRec := nsRecords(ns)
		sort.Sort(nsRec)
		p.wantNS = nsRec
	}
}

// CNAME applies the option that the prober wants specific CNAME record.
func CNAME(cname string) func(*DnsProber) {
	return func(p *DnsProber) {
		p.wantCNAME = cname
	}
}

// TXT applies the option that the prober wants specific TXT records.
func TXT(txt []string) func(*DnsProber) {
	return func(p *DnsProber) {
		sort.Strings(txt)
		p.wantTXT = txt
	}
}

// New returns a new instance of the DNS probe with specified options.
func New(target string, options ...func(*DnsProber)) *prober.Probe {
	p := &DnsProber{Target: target}
	for _, opt := range options {
		opt(p)
	}
	return prober.NewProbe(p, "DnsProber", fmt.Sprintf("Probes DNS records of %s", target),
		prober.Interval(time.Minute*5), prober.FailurePenalty(5))
}

// Probe verifies that the target's DNS records are as expected.
func (p *DnsProber) Probe() error {
	if len(p.wantMX) > 0 {
		glog.V(1).Infof("Checking %d MX records..\n", len(p.wantMX))
		err := p.checkMX()
		if err != nil {
			return err
		}
	}
	if len(p.wantA) > 0 {
		glog.V(1).Infof("Checking %d A records..\n", len(p.wantA))
		err := p.checkA()
		if err != nil {
			return err
		}
	}
	if len(p.wantNS) > 0 {
		glog.V(1).Infof("Checking %d NS records..\n", len(p.wantNS))
		err := p.checkNS()
		if err != nil {
			return err
		}
	}
	if p.wantCNAME != "" {
		glog.V(1).Infof("Checking CNAME record..\n")
		err := p.checkCNAME()
		if err != nil {
			return err
		}
	}
	if len(p.wantTXT) > 0 {
		glog.V(1).Infof("Checking %d TXT records..\n", len(p.wantTXT))
		err := p.checkTXT()
		if err != nil {
			return err
		}
	}
	return nil
}

// mxRecords is a collection of MX records, implementing sort.Interface.
//
// We need this custom order since the sort order in net.LookupMX
// randomizes records with the same preference value.
type mxRecords []*net.MX

func (r mxRecords) Len() int { return len(r) }

func (r mxRecords) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func (r mxRecords) Less(i, j int) bool {
	if r[i].Pref == r[j].Pref {
		return r[i].Host < r[j].Host
	}
	return r[i].Pref < r[j].Pref
}

// String returns a readable description of the MX records.
func (r mxRecords) String() string {
	s := ""
	for i, r := range r {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%s (%d)", r.Host, r.Pref)
	}
	return s
}

// checkMX verifies that the target has expected MX records.
func (p *DnsProber) checkMX() error {
	mx, err := net.LookupMX(p.Target)
	if err != nil {
		return fmt.Errorf("failed to look up MX records for %s: %v", p.Target, err)
	}
	mxRec := mxRecords(mx)
	if len(mxRec) != len(p.wantMX) {
		return fmt.Errorf("want %d MX records, got %d: %s", len(p.wantMX), len(mxRec), mxRec)
	}
	sort.Sort(mxRec)
	for i, r := range mxRec {
		if p.wantMX[i].Host != r.Host {
			return fmt.Errorf("bad host %q for MX record #%d; want %q", r.Host, i, p.wantMX[i].Host)
		}
		if p.wantMX[i].Pref != r.Pref {
			return fmt.Errorf("bad prio %d for MX record #%d; want %d", i, r.Pref, p.wantMX[i].Pref)
		}
	}
	return nil
}

// checkA verifies that the target has expected A records.
func (p *DnsProber) checkA() error {
	addr, err := net.LookupHost(p.Target)
	if err != nil {
		return fmt.Errorf("failed to look up A records for %s: %v", p.Target, err)
	}
	if len(addr) != len(p.wantA) {
		return fmt.Errorf("got %d A records, want %d", len(addr), len(p.wantA))
	}
	sort.Strings(addr)
	for i, a := range addr {
		if p.wantA[i] != a {
			return fmt.Errorf("bad A record %q at #%d; want %q", a, i, p.wantA[i])
		}
	}
	return nil
}

// nsRecords is a collection of NS records, implementing sort.Interface.
type nsRecords []*net.NS

func (r nsRecords) Len() int { return len(r) }

func (r nsRecords) Swap(i, j int) { r[i], r[j] = r[j], r[i] }

func (r nsRecords) Less(i, j int) bool { return r[i].Host < r[j].Host }

// String returns a readable description of the NS records.
func (ns nsRecords) String() string {
	s := ""
	for i, r := range ns {
		if i > 0 {
			s += ", "
		}
		s += r.Host
	}
	return s
}

// checkNS verifies that the target has expected NS records.
func (p *DnsProber) checkNS() error {
	ns, err := net.LookupNS(p.Target)
	if err != nil {
		return fmt.Errorf("failed to look up NS records for %s: %v", p.Target, err)
	}
	nsRec := nsRecords(ns)
	if len(nsRec) != len(p.wantNS) {
		return fmt.Errorf("want %d NS records, got %d: %s", len(p.wantNS), len(nsRec), nsRec)
	}
	sort.Sort(nsRec)
	for i, n := range ns {
		if p.wantNS[i].Host != n.Host {
			return fmt.Errorf("bad NS record %q at #%d; want %q", n, i, p.wantNS[i])
		}
	}
	return nil
}

// checkCNAME verifies that the target has expected CNAME record.
func (p *DnsProber) checkCNAME() error {
	cname, err := net.LookupCNAME(p.Target)
	if err != nil {
		return fmt.Errorf("failed to look up CNAME record for %s: %v", p.Target, err)
	}
	if cname != p.wantCNAME {
		return fmt.Errorf("bad CNAME record %q; want %q", cname, p.wantCNAME)
	}
	return nil
}

// checkTXT verifies that the target has expected TXT records.
func (p *DnsProber) checkTXT() error {
	txt, err := net.LookupTXT(p.Target)
	if err != nil {
		return err
	}
	if len(txt) != len(p.wantTXT) {
		return fmt.Errorf("want %d TXT records, got %d: %s", len(p.wantTXT), len(txt), txt)
	}
	sort.Strings(txt)
	for i, t := range txt {
		if p.wantTXT[i] != t {
			return fmt.Errorf("bad TXT record %q at #%d; want %q", t, i, p.wantTXT[i])
		}
	}
	return nil
}

// Alert sends an alert notification via email.
func (p *DnsProber) Alert(name, desc string, badness int, records prober.Records) error {
	return probes.SendAlertEmail(name, desc, badness, records)
}
