package rush

import (
	"net/http"
	"path"
	"slices"
	"strings"
)

type Middleware func(http.Handler) http.Handler

var allMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
}

type Router struct {
	// Configuration handlers
	NotFound              http.Handler
	MethodNotAllowed      http.Handler
	AutoOptions           http.Handler
	RedirectTrailingSlash bool

	// Internal state
	routes      *trie
	middlewares []Middleware
	prefix      string
	handler     http.Handler
	isRoot      bool
}

func New() *Router {
	return &Router{
		NotFound: http.NotFoundHandler(),
		MethodNotAllowed: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}),
		AutoOptions: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}),
		routes: &trie{root: newNode("/")},
		isRoot: true,
	}
}

func (r *Router) Use(middlewares ...Middleware) {
	if r.handler != nil {
		panic("rush: all root-level middlewares must be defined before routes")
	}
	r.middlewares = append(r.middlewares, middlewares...)
}

func (r *Router) cloneChain() []Middleware {
	if r.isRoot {
		return []Middleware{}
	}
	return slices.Clone(r.middlewares)
}

func (r *Router) Group(fn func(r *Router)) {
	sub := &Router{routes: r.routes, prefix: r.prefix, middlewares: r.cloneChain()}
	fn(sub)
}

func (r *Router) GroupWithPrefix(prefix string, fn func(r *Router)) {
	sub := &Router{routes: r.routes, prefix: r.prefix + prefix, middlewares: r.cloneChain()}
	fn(sub)
}

func (r *Router) With(middlewares ...Middleware) *Router {
	return &Router{
		routes:      r.routes,
		prefix:      r.prefix,
		middlewares: append(r.cloneChain(), middlewares...),
	}
}

func (r *Router) HandleFunc(pattern string, handler http.HandlerFunc, methods ...string) {
	r.Handle(pattern, handler, methods...)
}

func (r *Router) Handle(pattern string, handler http.Handler, methods ...string) {
	// Normalize method names to uppercase
	for i, m := range methods {
		methods[i] = strings.ToUpper(m)
	}

	// Auto-add HEAD method when GET is specified
	if slices.Contains(methods, http.MethodGet) && !slices.Contains(methods, http.MethodHead) {
		methods = append(methods, http.MethodHead)
	}

	// Default to all methods if none specified
	if len(methods) == 0 {
		methods = allMethods
	}

	// Build handler chain for root router
	if r.isRoot && r.handler == nil {
		r.handler = chain(r.middlewares, http.HandlerFunc(r.handleRequest))
	}

	// Apply middlewares for non-root routers
	if !r.isRoot {
		handler = chain(r.middlewares, handler)
	}

	r.routes.insert(r.prefix+pattern, handler, methods...)
}

func chain(middlewares []Middleware, handler http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
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

func (r *Router) Options(pattern string, handlerFunc http.HandlerFunc) {
	r.Handle(pattern, handlerFunc, http.MethodOptions)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, rq *http.Request) {
	if !r.isRoot {
		panic("rush: only root router should be used as http.Handler")
	}

	if r.handler == nil {
		r.handler = chain(r.middlewares, http.HandlerFunc(r.handleRequest))
	}

	r.handler.ServeHTTP(w, rq)
}

func needsCleaning(path string) bool {
	n := len(path) - 1
	if n > 1 && path[n] == '/' {
		return true
	}
	for i := range n {
		if path[i] == '/' && (path[i+1] == '/' || path[i+1] == '.') {
			return true
		}
	}
	return false
}

func (r *Router) handleRequest(w http.ResponseWriter, rq *http.Request) {
	urlPath := rq.URL.Path
	if needsCleaning(urlPath) {
		urlPath = path.Clean(urlPath)
	}

	match := r.routes.lookup(urlPath, rq)
	if match == nil {
		r.NotFound.ServeHTTP(w, rq)
		return
	}

	if r.RedirectTrailingSlash && urlPath != "/" && strings.HasSuffix(rq.URL.Path, "/") {
		code := http.StatusMovedPermanently
		if rq.Method != http.MethodGet {
			code = http.StatusPermanentRedirect
		}
		http.Redirect(w, rq, urlPath, code)
		return
	}

	handler, methodAllowed := match.handlers[rq.Method]
	if !methodAllowed {
		r.handleMethodNotAllowed(w, rq, match)
		return
	}
	handler.ServeHTTP(w, rq)
}

func (r *Router) handleMethodNotAllowed(w http.ResponseWriter, rq *http.Request, node *node) {
	w.Header().Set("Allow", node.allow())
	if rq.Method == http.MethodOptions {
		r.AutoOptions.ServeHTTP(w, rq)
	} else {
		r.MethodNotAllowed.ServeHTTP(w, rq)
	}
}
