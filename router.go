package rush

import (
	"net/http"
	"path"
	"slices"
	"strings"
)

type Middleware func(http.Handler) http.Handler

var AllMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
}

type Router struct {
	NotFound         http.Handler
	MethodNotAllowed http.Handler
	Options          http.Handler
	routes           *trie
	middlewares      []Middleware
	prefix           string
	handlersWrapped  bool // true once custom 404/405/OPTIONS handlers have been wrapped with middlewares
}

func New() *Router {
	return &Router{
		NotFound: http.NotFoundHandler(),
		MethodNotAllowed: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}),
		Options: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		routes: &trie{root: newNode("/")},
	}
}

func (r *Router) Use(mw ...Middleware) {
	r.middlewares = append(r.middlewares, mw...)
}

func (r *Router) Group(fn func(r *Router)) {
	nr := &Router{routes: r.routes, prefix: r.prefix, middlewares: slices.Clone(r.middlewares)}
	fn(nr)
}

func (r *Router) GroupWithPrefix(prefix string, fn func(r *Router)) {
	nr := &Router{
		routes:      r.routes,
		prefix:      r.prefix + prefix,
		middlewares: slices.Clone(r.middlewares),
	}
	fn(nr)
}

func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc, methods ...string) {
	r.Handle(pattern, handler, methods...)
}

func (r *Router) Handle(pattern string, handler http.Handler, methods ...string) {
	for i, m := range methods {
		methods[i] = strings.ToUpper(m)
	}

	if slices.Contains(methods, http.MethodGet) && !slices.Contains(methods, http.MethodHead) {
		methods = append(methods, http.MethodHead)
	}

	if len(methods) == 0 {
		methods = AllMethods
	}

	r.routes.insert(r.prefix+pattern, r.wrap(handler), methods...)
}

func (r *Router) Get(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodGet)
}

func (r *Router) Head(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodHead)
}

func (r *Router) Post(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodPost)
}

func (r *Router) Put(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodPut)
}

func (r *Router) Patch(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodPatch)
}

func (r *Router) Delete(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodDelete)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	urlPath := path.Clean(rq.URL.Path)
	r.wrapCustomHandlersOnce()

	match, found := r.routes.lookup(urlPath)
	if !found {
		r.NotFound.ServeHTTP(w, rq)
		return
	}

	handler, methodAllowed := match.node.handlers[rq.Method]
	if !methodAllowed {
		r.handleMethodNotAllowed(w, rq, match.node)
		return
	}

	for key, value := range match.params {
		rq.SetPathValue(key, value)
	}
	handler.ServeHTTP(w, rq)
}

func (r *Router) wrap(h http.Handler) http.Handler {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

func (r *Router) wrapCustomHandlersOnce() {
	if !r.handlersWrapped {
		r.NotFound = r.wrap(r.NotFound)
		r.MethodNotAllowed = r.wrap(r.MethodNotAllowed)
		r.Options = r.wrap(r.Options)
		r.handlersWrapped = true
	}
}

func (r *Router) handleMethodNotAllowed(w http.ResponseWriter, rq *http.Request, node *node) {
	w.Header().Set("Allow", node.allow())
	if rq.Method == http.MethodOptions {
		r.Options.ServeHTTP(w, rq)
	} else {
		r.MethodNotAllowed.ServeHTTP(w, rq)
	}
}
