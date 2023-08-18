// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016 Datadog, Inc.

// Package gearbox provides functions to trace the gogearbox/gearbox package (https://github.com/gogearbox/gearbox)
package gearbox // import "gopkg.in/DataDog/dd-trace-go.v1/contrib/gogearbox/gearbox"

import (
	"fmt"
	"strconv"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/gogearbox/gearbox.v1/internal/gearboxutil"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/log"
	"gopkg.in/DataDog/dd-trace-go.v1/internal/telemetry"

	"github.com/gogearbox/gearbox"
	"github.com/valyala/fasthttp"
)

const componentName = "gogearbox/gearbox.v1"

func init() {
	telemetry.LoadIntegration(componentName)
}

// Middleware returns middleware that will trace incoming requests.
func Middleware(opts ...Option) func(gctx gearbox.Context) {
	cfg := newConfig()
	for _, fn := range opts {
		fn(cfg)
	}
	log.Debug("contrib/gogearbox/gearbox: Configuring Middleware: Service: %#v", cfg)
	spanOpts := []tracer.StartSpanOption{
		tracer.ServiceName(cfg.serviceName),
	}
	return func(gctx gearbox.Context) {
		if cfg.ignoreRequest(gctx) {
			gctx.Next()
			return
		}
		fctx := gctx.Context()
		spanOpts = defaultSpanTags(spanOpts, fctx)
		// Create an instance of FasthttpCarrier, which embeds *fasthttp.RequestCtx and implements TextMapReader
		fcc := &gearboxutil.FasthttpCarrier{
			ReqHeader: &fctx.Request.Header,
		}
		if sctx, err := tracer.Extract(fcc); err == nil {
			spanOpts = append(spanOpts, tracer.ChildOf(sctx))
		}
		span, _ := tracer.StartSpanFromContext(fctx, "http.request", spanOpts...)
		defer span.Finish()

		// AFAICT, there is no automatic way to update the fashttp context with the context returned from tracer.StartSpanFromContext
		// Instead I had to manually add the activeSpanKey onto the fashttp context
		activeSpanKey := tracer.ContextKey{}
		fctx.SetUserValue(activeSpanKey, span)

		gctx.Next()

		span.SetTag(ext.ResourceName, cfg.resourceNamer(gctx))

		status := fctx.Response.StatusCode()
		if cfg.isStatusError(status) {
			span.SetTag(ext.Error, fmt.Errorf("%d: %s", status, string(fctx.Response.Body())))
		}
		span.SetTag(ext.HTTPCode, strconv.Itoa(status))
	}
}

// MTOFF: Does it matter when these span tags are added?
// other integrations have some tags added after startSpan/before FinishSpan,
// whereas I'm adding as many as possible before startSpan, since none of these depend on operations that happen further down the req chain AFAICT
func defaultSpanTags(opts []tracer.StartSpanOption, ctx *fasthttp.RequestCtx) []tracer.StartSpanOption {
	opts = append([]ddtrace.StartSpanOption{
		tracer.Tag(ext.Component, componentName),
		tracer.Tag(ext.SpanKind, ext.SpanKindServer),
		tracer.SpanType(ext.SpanTypeWeb),
		tracer.Tag(ext.HTTPMethod, string(ctx.Method())),
		tracer.Tag(ext.HTTPURL, string(ctx.URI().FullURI())),
		tracer.Tag(ext.HTTPUserAgent, string(ctx.UserAgent())),
		tracer.Measured(),
	}, opts...)
	if host := string(ctx.Host()); len(host) > 0 {
		opts = append([]ddtrace.StartSpanOption{tracer.Tag("http.host", host)}, opts...)
	}
	return opts
}
