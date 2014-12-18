package dashboard

import (
	"flag"
	"html/template"
	"net/http"
	"os"

	// Generated with `go-bindata -pkg="bindata" -o bindata/bin.go tmpl/`
	// from the base directory.

	"hkjn.me/dashboard/bindata"
	"hkjn.me/googleauth"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	authDisabled = flag.Bool("no_auth", false, "disables authentication (use for testing only)")
	baseTmpls    = []string{
		"tmpl/base.tmpl",
		"tmpl/scripts.tmpl",
		"tmpl/style.tmpl",
	}
	indexTmpls = append(
		baseTmpls,
		"tmpl/index.tmpl",
		"tmpl/links.tmpl",
		"tmpl/prober.tmpl",
	)
	configMissingTmpl = getTemplate(append(
		baseTmpls,
		"tmpl/config_missing.tmpl",
	))
	baseTemplate = "base"
)

// newRouter returns a new router for the endpoints of the dashboard.
//
// newRouter panics if the config wasn't loaded.
func newRouter() *mux.Router {
	routes := []route{
		newPage("/", indexTmpls, getIndexData),
		simpleRoute{"/connect", "GET", googleauth.ConnectHandler},
	}

	router := mux.NewRouter().StrictSlash(true)
	for _, r := range routes {
		glog.V(1).Infof("Registering route for %q on %q\n", r.Method(), r.Pattern())
		router.
			Methods(r.Method()).
			Path(r.Pattern()).
			HandlerFunc(r.HandlerFunc())
	}
	return router
}

// getTemplate returns the template loaded from the paths.
//
// getTemplate uses the bindata package on live, and otherwise parses
// the .tmpl files from disk.
func getTemplate(tmpls []string) *template.Template {
	live := true
	if cfg.loaded { // TODO: improve this hack.
		live = cfg.Live
	}
	glog.Infof("we're live? %v\n", live)
	if live {
		assets := []byte{}
		for _, t := range tmpls {
			b, err := bindata.Asset(t)
			if err != nil {
				glog.Fatalf("can't load asset %q: %v\n", t, err)
			}
			assets = append(assets, b...)
		}
		return template.Must(template.New(baseTemplate).Parse(string(assets)))
	}
	// TODO: automatically rebuild bindata (and gofmt -w it, since it
	// isn't competent enough to do that..) in deploy.sh.
	// Read from local disk on dev.
	return template.Must(template.ParseFiles(tmpls...))
}

// serveISE serves an internal server error to the user.
func serveISE(w http.ResponseWriter) {
	http.Error(w, "Internal server error.", http.StatusInternalServerError)
}

// route describes how to serve HTTP on an endpoint.
type route interface {
	Method() string                // GET, POST, PUT, etc.
	Pattern() string               // URI for the route
	HandlerFunc() http.HandlerFunc // HTTP handler func
}

// simpleRoute implements the route interface for endpoints.
type simpleRoute struct {
	pattern, method string
	handlerFunc     http.HandlerFunc
}

func (r simpleRoute) Method() string { return r.method }

func (r simpleRoute) Pattern() string { return r.pattern }

func (r simpleRoute) HandlerFunc() http.HandlerFunc { return r.handlerFunc }

// getDataFn is a function to get template data.
type getDataFn func(http.ResponseWriter, *http.Request) (interface{}, error)

// page implements the route interface for endpoints that render HTML.
type page struct {
	pattern         string
	tmpl            *template.Template // backing template
	getTemplateData getDataFn
}

// newPage returns a new page.
func newPage(pattern string, tmpls []string, getData getDataFn) *page {
	return &page{
		pattern,
		getTemplate(tmpls),
		getData,
	}
}

func (p page) Method() string { return "GET" }

func (p page) Pattern() string { return p.pattern }

// HandlerFunc returns the http handler func, which renders the
// template with the data.
func (p page) HandlerFunc() http.HandlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		data, err := p.getTemplateData(w, r)
		if err != nil {
			glog.Errorf("error getting template data: %v\n", err)
			serveISE(w)
			return
		}
		err = p.tmpl.ExecuteTemplate(w, baseTemplate, data)
		if err != nil {
			glog.Errorf("error rendering template: %v\n", err)
			serveISE(w)
			return
		}
	}

	if !flag.Parsed() {
		flag.Parse()
	}
	if *authDisabled {
		glog.Infof("-disabled_auth is set, not checking credentials\n")
	} else {
		fn = googleauth.RequireLogin(fn)
	}
	return checkConfig(fn)
}

// checkConfig returns a wrapped HandlerFunc that attempts to load
// config.yaml if it hasn't been loaded, and calls the specified
// HandlerFunc only if there is a config.
func checkConfig(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if cfg.Loaded() {
			glog.V(1).Infoln("already have config")
			fn(w, r)
			return
		}
		err := cfg.Load()
		if err != nil {
			if os.IsNotExist(err) {
				glog.V(1).Infof("no config: %v\n", err)
				err = configMissingTmpl.ExecuteTemplate(w, baseTemplate, "")
				if err != nil {
					glog.Errorf("failed to execute 'no config' template: %v\n", err)
					http.Error(w, "Internal server error.", http.StatusInternalServerError)
					return
				}
			} else {
				glog.Errorf("bad config: %v\n", err)
				http.Error(w, "Internal server error.", http.StatusInternalServerError)
			}
			return
		}
		glog.V(1).Infoln("config loaded successfully, onward to original handler func")
		fn(w, r)
	}
}
