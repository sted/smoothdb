package server

import (
	"context"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type Params = httprouter.Params
type Handler func(context.Context, ResponseWriter, *Request)
type Middleware func(Handler) Handler

type Router struct {
	*httprouter.Router
	server *Server
}

type Group struct {
	base *Router
	path string
}

func NewRouter(server *Server) *Router {
	return &Router{httprouter.New(), server}
}

func (router *Router) Group(path string) *Group {
	return &Group{router, path}
}

func (router *Router) Handle(method string, path string, handler Handler) {
	router.Router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		rw := &responseWriter{ResponseWriter: w}
		req := &Request{Request: r, params: p}
		handler(r.Context(), rw, req)
	})
}

func (router *Router) HandleWithDb(method string, path string, handler Handler) {
	router.Router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		rw := &responseWriter{ResponseWriter: w}
		req := &Request{Request: r, params: p}
		DatabaseAccess(rw, req, router.server, false, handler)
	})
}

func Adapt(h http.HandlerFunc) Handler {
	return func(c context.Context, w ResponseWriter, r *Request) {
		h(w, r.Request)
	}
}

func (g *Group) Group(path string) *Group {
	return &Group{g.base, path}
}

func (g *Group) Handle(method string, path string, handler Handler) {
	g.base.Handle(method, g.path+path, handler)
}

func (g *Group) HandleWithDb(method string, path string, handler Handler) {
	g.base.HandleWithDb(method, g.path+path, handler)
}
