package dashboard

import (
	"flag"
	"html/template"
	"net/http"
	"sync"

	"hkjn.me/googleauth"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	authDisabled   = flag.Bool("no_auth", false, "disables authentication (use for testing only)")
	parseFlagsOnce = sync.Once{}
	indexTmpls     = []string{
		"tmpl/base.tmpl",
		"tmpl/scripts.tmpl",
		"tmpl/style.tmpl",
		"tmpl/index.tmpl",
		"tmpl/links.tmpl",
		"tmpl/prober.tmpl",
	}
	routes = []route{
		newPage("/", indexTmpls, getIndexData),
		simpleRoute{"/connect", "GET", googleauth.ConnectHandler},
	}
)

// NewRouter returns a new router.
func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)
	for _, r := range routes {
		glog.V(1).Infof("registering route for %q on %q\n", r.Method(), r.Pattern())
		router.
			Methods(r.Method()).
			Path(r.Pattern()).
			HandlerFunc(r.HandlerFunc())
	}
	return router
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
	method, pattern string
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
		template.Must(template.ParseFiles(tmpls...)),
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
		err = p.tmpl.ExecuteTemplate(w, "base", data)
		if err != nil {
			glog.Errorf("error rendering template: %v\n", err)
			serveISE(w)
			return
		}
	}

	parseFlagsOnce.Do(func() {
		if !flag.Parsed() {
			flag.Parse()
		}
	})
	if *authDisabled {
		glog.Infof("-disabled_auth is set, not checking credentials\n")
	} else {
		fn = googleauth.RequireLogin(fn)
	}
	return fn
}
