package dotweb

import (
	"github.com/devfeel/dotweb/framework/convert"
	"github.com/devfeel/dotweb/logger"
	"time"
)

const (
	middleware_App    = "app"
	middleware_Group  = "group"
	middleware_Router = "router"
)

type MiddlewareFunc func() Middleware

//middleware执行优先级：
//优先级1：app级别middleware
//优先级2：group级别middleware
//优先级3：router级别middleware

// Middleware middleware interface
type Middleware interface {
	Handle(ctx Context) error
	SetNext(m Middleware)
	Next(ctx Context) error
	Exclude(routers ...string)
}

//middleware 基础类，应用可基于此实现完整Moddleware
type BaseMiddlware struct {
	next           Middleware
	excludeRouters map[string]struct{}
}

func (bm *BaseMiddlware) SetNext(m Middleware) {
	bm.next = m
}

func (bm *BaseMiddlware) Next(ctx Context) error {
	httpCtx := ctx.(*HttpContext)
	if httpCtx.middlewareStep == "" {
		httpCtx.middlewareStep = middleware_App
	}
	if bm.next == nil {
		if httpCtx.middlewareStep == middleware_App {
			httpCtx.middlewareStep = middleware_Group
			if len(httpCtx.RouterNode().GroupMiddlewares()) > 0 {
				return httpCtx.RouterNode().GroupMiddlewares()[0].Handle(ctx)
			}
		}
		if httpCtx.middlewareStep == middleware_Group {
			httpCtx.middlewareStep = middleware_Router
			if len(httpCtx.RouterNode().Middlewares()) > 0 {
				return httpCtx.RouterNode().Middlewares()[0].Handle(ctx)
			}
		}

		if httpCtx.middlewareStep == middleware_Router {
			return httpCtx.Handler()(ctx)
		}
	} else {
		return bm.next.Handle(ctx)
	}
	return nil
}

// Exclude Exclude this middleware with router
func (bm *BaseMiddlware) Exclude(routers ...string) {
	if bm.excludeRouters == nil {
		bm.excludeRouters = make(map[string]struct{})
	}
	for _, v := range routers {
		if _, exists := bm.excludeRouters[v]; !exists {
			bm.excludeRouters[v] = struct{}{}
		}
	}
}

type xMiddleware struct {
	BaseMiddlware
	IsEnd bool
}

func (x *xMiddleware) Handle(ctx Context) error {
	httpCtx := ctx.(*HttpContext)
	if httpCtx.middlewareStep == "" {
		httpCtx.middlewareStep = middleware_App
	}
	if x.IsEnd {
		return httpCtx.Handler()(ctx)
	}
	return x.Next(ctx)
}

//请求日志中间件
type RequestLogMiddleware struct {
	BaseMiddlware
}

func (m *RequestLogMiddleware) Handle(ctx Context) error {
	m.Next(ctx)
	timetaken := int64(time.Now().Sub(ctx.(*HttpContext).startTime) / time.Millisecond)
	log := ctx.Request().Url() + " " + logContext(ctx, timetaken)
	logger.Logger().Debug(log, LogTarget_HttpRequest)
	return nil
}

//get default log string
func logContext(ctx Context, timetaken int64) string {
	var reqbytelen, resbytelen, method, proto, status, userip string
	if ctx != nil {
		reqbytelen = convert.Int642String(ctx.Request().ContentLength)
		resbytelen = convert.Int642String(ctx.Response().Size)
		method = ctx.Request().Method
		proto = ctx.Request().Proto
		status = convert.Int2String(ctx.Response().Status)
		userip = ctx.RemoteIP()
	}

	log := method + " "
	log += userip + " "
	log += proto + " "
	log += status + " "
	log += reqbytelen + " "
	log += resbytelen + " "
	log += convert.Int642String(timetaken)

	return log
}
