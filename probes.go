package dashboard

import (
	"flag"
	"net"
	"sync"

	"github.com/golang/glog"

	"hkjn.me/prober"
	"hkjn.me/probes/dnsprobe"
	"hkjn.me/probes/webprobe"
)

var (
	proberDisabled = flag.Bool("no_probes", false, "disables probes")
	allProbes      = []*prober.Probe{}
	createOnce     = sync.Once{}
)

func getWebProbes() []*prober.Probe {
	probes := []*prober.Probe{}
	for _, p := range probecfg.WebProbes {
		probes = append(probes,
			webprobe.New(
				p.Target,
				"GET",
				p.WantStatus,
				webprobe.Name(p.Name),
				webprobe.InResponse(p.Want)))
	}
	return probes
}

func getDnsProbe() *prober.Probe {
	mxRecords := []*net.MX{}
	r := probecfg.DnsProbe.Records
	for _, mx := range r.Mx {
		mxRecords = append(mxRecords, &net.MX{mx.Host, mx.Pref})
	}
	nsRecords := []*net.NS{}
	for _, ns := range r.Ns {
		nsRecords = append(nsRecords, &net.NS{ns})
	}
	return dnsprobe.New(
		probecfg.DnsProbe.Target, dnsprobe.MX(mxRecords), dnsprobe.A(r.A),
		dnsprobe.NS(nsRecords), dnsprobe.CNAME(r.Cname), dnsprobe.TXT(r.Txt))
}

// getProbes returns all probes in the dashboard.
func getProbes() []*prober.Probe {
	createOnce.Do(func() {
		if !flag.Parsed() {
			flag.Parse()
		}
		if *proberDisabled {
			glog.Infof("Probes are disabled with -no_probes\n")
		} else {
			allProbes = []*prober.Probe{
				getDnsProbe(),
			}
			allProbes = append(allProbes, getWebProbes()...)
		}
	})
	return allProbes
}
