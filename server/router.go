package server

// import (
// 	"context"
// 	"net/http"
// 	"unsafe"

// 	"github.com/julienschmidt/httprouter"
// )

// type Params = httprouter.Params
// type Handler func(context.Context, ResponseWriter, *Request)
// type Middleware func(Handler) Handler

// type Router struct {
// 	*httprouter.Router
// 	middlewares []Middleware
// }

// type Group struct {
// 	base        *Router
// 	path        string
// 	middlewares []Middleware
// }

// func NewRouter() *Router {
// 	return &Router{httprouter.New(), nil}
// }

// func (router *Router) Use(middlewares ...Middleware) {
// 	router.middlewares = append(router.middlewares, middlewares...)
// }

// func (router *Router) Group(path string, middlewares ...Middleware) *Group {
// 	return &Group{router, path, append(router.middlewares, middlewares...)}
// }

// //go:nosplit
// func noescape(p unsafe.Pointer) unsafe.Pointer {
// 	x := uintptr(p)
// 	return unsafe.Pointer(x ^ 0)
// }

// func (router *Router) Handle(method string, path string, handler Handler) {
// 	handler = chain(handler, router.middlewares)
// 	router.Router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
// 		rw := noescape(unsafe.Pointer(&responseWriter{ResponseWriter: w}))
// 		req := noescape(unsafe.Pointer(&Request{Request: r, params: p}))
// 		handler(r.Context(), (*responseWriter)(rw), (*Request)(req))
// 	})
// }

// type Handler2 func(context.Context, Response, RequestR)

//	func (router *Router) Handle2(method string, path string, handler Handler2) {
//		//handler = chain(handler, router.middlewares)
//		router.Router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
//			//rr := RequestResponse{writer: w, request: r, params: p}
//			handler(r.Context(), Response{writer: w}, RequestR{request: r, params: p})
//		})
//	}

// func (g *Group) Group(path string, middlewares ...Middleware) *Group {
// 	return &Group{g.base, g.path + path, append(g.middlewares, middlewares...)}
// }

// func (g *Group) Handle(method string, path string, handler Handler) {
// 	handler = chain(handler, g.middlewares)
// 	g.base.Handle(method, g.path+path, handler)
// }

// func chain(h Handler, middlewares []Middleware) Handler {
// 	for i := len(middlewares) - 1; i >= 0; i-- {
// 		h = middlewares[i](h)
// 	}
// 	return h
// }
