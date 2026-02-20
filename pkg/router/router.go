package router

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
)

type Middleware func(http.Handler) http.Handler

type Router struct {
	mux    chi.Router
	routes map[string]string
	mu     sync.RWMutex
}

type Group struct {
	router      *Router
	prefix      string
	middlewares []Middleware
}

func New() *Router {
	return &Router{
		mux:    chi.NewRouter(),
		routes: make(map[string]string),
	}
}

func (r *Router) Handler() http.Handler {
	return r.mux
}

func (r *Router) Use(middlewares ...Middleware) {
	for _, mw := range middlewares {
		r.mux.Use(mw)
	}
}

func (r *Router) Group(prefix string, middlewares ...Middleware) *Group {
	return &Group{
		router:      r,
		prefix:      normalizePath(prefix),
		middlewares: append([]Middleware(nil), middlewares...),
	}
}

func (r *Router) Get(path, name string, handler http.HandlerFunc, middlewares ...Middleware) {
	r.mount(http.MethodGet, path, name, handler, middlewares...)
}

func (r *Router) Post(path, name string, handler http.HandlerFunc, middlewares ...Middleware) {
	r.mount(http.MethodPost, path, name, handler, middlewares...)
}

func (r *Router) Path(name string) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	path, ok := r.routes[name]
	return path, ok
}

func (r *Router) URL(name string, params map[string]string) (string, error) {
	path, ok := r.Path(name)
	if !ok {
		return "", fmt.Errorf("route %q not found", name)
	}

	for key, value := range params {
		path = strings.ReplaceAll(path, "{"+key+"}", value)
	}

	if strings.Contains(path, "{") {
		return "", fmt.Errorf("missing parameters for route %q", name)
	}

	return path, nil
}

func (r *Router) mount(method, path, name string, handler http.HandlerFunc, middlewares ...Middleware) {
	fullPath := normalizePath(path)
	h := chain(handler, middlewares...)
	r.mux.Method(method, fullPath, h)

	if name == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.routes[name] = fullPath
}

func (g *Group) Group(prefix string, middlewares ...Middleware) *Group {
	joined := joinPath(g.prefix, prefix)
	combined := append(append([]Middleware(nil), g.middlewares...), middlewares...)

	return &Group{
		router:      g.router,
		prefix:      joined,
		middlewares: combined,
	}
}

func (g *Group) Get(path, name string, handler http.HandlerFunc, middlewares ...Middleware) {
	g.mount(http.MethodGet, path, name, handler, middlewares...)
}

func (g *Group) Post(path, name string, handler http.HandlerFunc, middlewares ...Middleware) {
	g.mount(http.MethodPost, path, name, handler, middlewares...)
}

func (g *Group) mount(method, path, name string, handler http.HandlerFunc, middlewares ...Middleware) {
	fullPath := joinPath(g.prefix, path)
	combined := append(append([]Middleware(nil), g.middlewares...), middlewares...)
	h := chain(handler, combined...)

	g.router.mux.Method(method, fullPath, h)

	if name == "" {
		return
	}

	g.router.mu.Lock()
	defer g.router.mu.Unlock()
	g.router.routes[name] = fullPath
}

func chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	if len(middlewares) == 0 {
		return handler
	}

	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	return wrapped
}

func joinPath(parts ...string) string {
	if len(parts) == 0 {
		return "/"
	}

	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.Trim(part, "/")
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}

	if len(segments) == 0 {
		return "/"
	}

	return "/" + strings.Join(segments, "/")
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	return joinPath(path)
}
