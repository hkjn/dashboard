package dashboard

import (
	"net/http"

	"hkjn.me/prober"
)

// linkInfo describes
type linkInfo struct {
	Name, URL string
}

func getIndexData(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	data := struct {
		ErrorMsg       string
		Links          []linkInfo
		Probes         []*prober.Probe
		ProberDisabled bool
	}{}
	data.Links = []linkInfo{
	//		linkInfo{"TODO", "/fixme_link_to_dashboard_goes_here"},
	}
	data.Probes = GetProbes()
	data.ProberDisabled = *proberDisabled
	return data, nil
}
