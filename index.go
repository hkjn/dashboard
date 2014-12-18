package dashboard

import (
	"net/http"

	"hkjn.me/prober"
)

// getIndexData returns the data for the index page.
func getIndexData(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	data := struct {
		Version string
		Links   []struct {
			Name, URL string
		}
		Probes         []*prober.Probe
		ProberDisabled bool
	}{}
	data.Version = cfg.Version
	data.Probes = getProbes()
	data.ProberDisabled = *proberDisabled
	return data, nil
}
