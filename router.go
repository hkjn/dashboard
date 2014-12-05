package dashboard

import (
	"html/template"
	"net/http"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

var (
	indexTmpls = []string{
		"tmpl/base.tmpl",
		"tmpl/scripts.tmpl",
		"tmpl/style.tmpl",
		"tmpl/index.tmpl",
		"tmpl/links.tmpl",
		"tmpl/prober.tmpl",
		"tmpl/jquery.tmpl",
	}
	routes = []route{
		newPage("/",
			[]string{indexTmpls},
			getIndexData,
		),
		simpleRoute{
			"/connect",
			"GET",
			connect,
		},
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

type route interface {
	Method() string
	Pattern() string
	HandlerFunc() handlerFunc
}

// handlerFunc is a local alias of http.HandlerFunc to allow extra methods.
type handlerFunc http.HandlerFunc

// simpleRoute implements the route interface for simple endpoints.
// TODO: better naming.
type simpleRoute struct {
	pattern, method string
	handlerFunc     handlerFunc
}

func (r simpleRoute) Method() string { return r.method }

func (r simpleRoute) Pattern() string { return r.pattern }

func (r simpleRoute) HandlerFunc() handlerFunc { return r.handlerFunc }

// renderFunc is a function to render a page.
type renderFunc func(http.ResponseWriter, *http.Request) (interface{}, error)

// page implements the route interface for endpoints that render HTML.
type page struct {
	pattern string
	tmpl    *template.Template // backing template
	render  renderFunc
}

var loginTmpl = template.Must(template.ParseFiles("tmpl/login.tmpl", "tmpl/jquery.tmpl"))

func newPage(pattern string, tmpls []string, renderFn renderFunc) *page {
	return &page{
		pattern,
		template.Must(template.ParseFiles(tmpls...)),
		renderFn,
	}
}

func (p page) Method() string { return "GET" }

func (p page) Pattern() string { return p.pattern }

func (p page) HandlerFunc() handlerFunc {
	fn := func(w http.ResponseWriter, r *http.Request) {
		data, err := p.render(w, r)
		if err != nil {
			serveISE(w)
			return
		}
		err = p.tmpl.ExecuteTemplate(w, "base", data)
		if err != nil {
			serveISE(w)
			return
		}
	}
	return handlerFunc(fn).requireLogin()
}
