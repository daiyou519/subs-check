// Package router provides a reflection-based routing system for the Gin web framework,
// allowing declarative route definitions through Go structs and reflection.
//
// This package simplifies route registration by leveraging Go's reflection capabilities
// to automatically register handlers from struct methods, supporting both individual
// routes and route groups with shared middlewares.
package router

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// Method represents an HTTP method.
type Method string

// HTTP methods supported by the router.
const (
	GET     Method = "GET"
	POST    Method = "POST"
	PUT     Method = "PUT"
	DELETE  Method = "DELETE"
	HEAD    Method = "HEAD"
	OPTIONS Method = "OPTIONS"
	PATCH   Method = "PATCH"
	ANY     Method = "ANY"
)

// ValidationError represents an error that occurs during route validation.
type ValidationError struct {
	Message string
}

// Error returns the error message.
func (e ValidationError) Error() string {
	return e.Message
}

// GroupRouter defines a group of routes sharing the same path prefix and middlewares.
type GroupRouter struct {
	Path        string
	Routes      []*Route
	Middlewares []gin.HandlerFunc
}

// NewGroupRouter creates a new GroupRouter with the given path.
func NewGroupRouter(path string) *GroupRouter {
	return &GroupRouter{
		Path:   path,
		Routes: make([]*Route, 0),
	}
}

// Use adds middlewares to the group.
func (g *GroupRouter) Use(middlewares ...gin.HandlerFunc) *GroupRouter {
	g.Middlewares = append(g.Middlewares, middlewares...)
	return g
}

// AddRoute adds a route to the group.
func (g *GroupRouter) AddRoute(route *Route) *GroupRouter {
	g.Routes = append(g.Routes, route)
	return g
}

// Route defines a single endpoint with its handlers and middlewares.
type Route struct {
	Path        string
	Method      Method
	Handlers    []gin.HandlerFunc
	Middlewares []gin.HandlerFunc
	Description string
}

// NewRoute creates a new Route instance with the given path and method.
func NewRoute(path string, method Method) *Route {
	return &Route{
		Path:     path,
		Method:   method,
		Handlers: make([]gin.HandlerFunc, 0),
	}
}

// Handle adds handler functions to the route.
func (r *Route) Handle(handlers ...gin.HandlerFunc) *Route {
	r.Handlers = append(r.Handlers, handlers...)
	return r
}

// Use adds middlewares to the route.
func (r *Route) Use(middlewares ...gin.HandlerFunc) *Route {
	r.Middlewares = append(r.Middlewares, middlewares...)
	return r
}

// WithDescription adds a description to the route.
func (r *Route) WithDescription(description string) *Route {
	r.Description = description
	return r
}

// Validate checks if the route is properly configured.
func (r *Route) Validate() error {
	if r.Path == "" {
		return ValidationError{"Route path cannot be empty"}
	}

	if len(r.Handlers) == 0 {
		return ValidationError{"Route must have at least one handler"}
	}

	return nil
}

// Router is an interface that defines the contract for a router implementation.
type Router interface {
	Routes() []*Route
}

// GroupedRouter is an interface that defines the contract for a grouped router implementation.
type GroupedRouter interface {
	Groups() []*GroupRouter
}

// Register registers routes from router methods to the Gin engine.
// The router parameter should be a struct with methods returning *Route.
func Register(engine *gin.Engine, router interface{}) error {
	routerType := reflect.TypeOf(router)
	routerValue := reflect.ValueOf(router)

	if r, ok := router.(Router); ok {
		return registerDirectRoutes(engine, r.Routes())
	}

	for i := 0; i < routerType.NumMethod(); i++ {
		method := routerType.Method(i)

		if method.Type.NumOut() != 1 || method.Type.Out(0) != reflect.TypeOf(&Route{}) {
			continue
		}

		result := routerValue.Method(i).Call(nil)
		if len(result) != 1 {
			continue
		}

		route, ok := result[0].Interface().(*Route)
		if !ok || route == nil {
			continue
		}

		if err := route.Validate(); err != nil {
			fnName := runtime.FuncForPC(method.Func.Pointer()).Name()
			return fmt.Errorf("invalid route from %s: %w", fnName, err)
		}

		allHandlers := make([]gin.HandlerFunc, 0, len(route.Middlewares)+len(route.Handlers))
		allHandlers = append(allHandlers, route.Middlewares...)
		allHandlers = append(allHandlers, route.Handlers...)

		registerRoute(engine, route.Method, route.Path, allHandlers)
	}

	return nil
}

// registerDirectRoutes registers routes directly from a slice of routes.
func registerDirectRoutes(engine *gin.Engine, routes []*Route) error {
	for _, route := range routes {
		if err := route.Validate(); err != nil {
			return err
		}

		allHandlers := make([]gin.HandlerFunc, 0, len(route.Middlewares)+len(route.Handlers))
		allHandlers = append(allHandlers, route.Middlewares...)
		allHandlers = append(allHandlers, route.Handlers...)

		registerRoute(engine, route.Method, route.Path, allHandlers)
	}

	return nil
}

// RegisterGroup registers route groups from router methods to the Gin engine.
// The router parameter should be a struct with methods returning *GroupRouter.
func RegisterGroup(engine *gin.Engine, router interface{}) error {
	routerType := reflect.TypeOf(router)
	routerValue := reflect.ValueOf(router)

	if r, ok := router.(GroupedRouter); ok {
		return registerDirectGroups(engine, r.Groups())
	}

	for i := 0; i < routerType.NumMethod(); i++ {
		method := routerType.Method(i)

		if method.Type.NumOut() != 1 || method.Type.Out(0) != reflect.TypeOf(&GroupRouter{}) {
			continue
		}

		result := routerValue.Method(i).Call(nil)
		if len(result) != 1 {
			continue
		}

		groupRouter, ok := result[0].Interface().(*GroupRouter)
		if !ok || groupRouter == nil {
			continue
		}

		for _, route := range groupRouter.Routes {
			if err := route.Validate(); err != nil {
				fnName := runtime.FuncForPC(method.Func.Pointer()).Name()
				return fmt.Errorf("invalid route in group %s from %s: %w",
					groupRouter.Path, fnName, err)
			}
		}

		group := engine.Group(groupRouter.Path, groupRouter.Middlewares...)

		for _, route := range groupRouter.Routes {
			allHandlers := make([]gin.HandlerFunc, 0, len(route.Middlewares)+len(route.Handlers))
			allHandlers = append(allHandlers, route.Middlewares...)
			allHandlers = append(allHandlers, route.Handlers...)

			registerRouteToGroup(group, route.Method, route.Path, allHandlers)
		}
	}

	return nil
}

// registerDirectGroups registers groups directly from a slice of group routers.
func registerDirectGroups(engine *gin.Engine, groups []*GroupRouter) error {
	for _, groupRouter := range groups {
		for _, route := range groupRouter.Routes {
			if err := route.Validate(); err != nil {
				return fmt.Errorf("invalid route in group %s: %w", groupRouter.Path, err)
			}
		}

		group := engine.Group(groupRouter.Path, groupRouter.Middlewares...)

		for _, route := range groupRouter.Routes {
			allHandlers := make([]gin.HandlerFunc, 0, len(route.Middlewares)+len(route.Handlers))
			allHandlers = append(allHandlers, route.Middlewares...)
			allHandlers = append(allHandlers, route.Handlers...)

			registerRouteToGroup(group, route.Method, route.Path, allHandlers)
		}
	}

	return nil
}

// MustRegister is like Register but panics if an error occurs.
func MustRegister(engine *gin.Engine, router interface{}) {
	if err := Register(engine, router); err != nil {
		panic(err)
	}
}

// MustRegisterGroup is like RegisterGroup but panics if an error occurs.
func MustRegisterGroup(engine *gin.Engine, router interface{}) {
	if err := RegisterGroup(engine, router); err != nil {
		panic(err)
	}
}

// registerRoute registers a single route to the Gin engine.
func registerRoute(engine *gin.Engine, method Method, path string, handlers []gin.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	switch method {
	case GET:
		engine.GET(path, handlers...)
	case POST:
		engine.POST(path, handlers...)
	case PUT:
		engine.PUT(path, handlers...)
	case DELETE:
		engine.DELETE(path, handlers...)
	case HEAD:
		engine.HEAD(path, handlers...)
	case OPTIONS:
		engine.OPTIONS(path, handlers...)
	case PATCH:
		engine.PATCH(path, handlers...)
	case ANY:
		engine.Any(path, handlers...)
	default:
		engine.GET(path, handlers...)
	}
}

// registerRouteToGroup registers a single route to a Gin route group.
func registerRouteToGroup(group *gin.RouterGroup, method Method, path string, handlers []gin.HandlerFunc) {
	if len(handlers) == 0 {
		return
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	switch method {
	case GET:
		group.GET(path, handlers...)
	case POST:
		group.POST(path, handlers...)
	case PUT:
		group.PUT(path, handlers...)
	case DELETE:
		group.DELETE(path, handlers...)
	case HEAD:
		group.HEAD(path, handlers...)
	case OPTIONS:
		group.OPTIONS(path, handlers...)
	case PATCH:
		group.PATCH(path, handlers...)
	case ANY:
		group.Any(path, handlers...)
	default:
		group.GET(path, handlers...)
	}
}
