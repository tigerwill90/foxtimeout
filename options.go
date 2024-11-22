// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxtimeout/blob/master/LICENSE.txt.

package foxtimeout

import (
	"github.com/tigerwill90/fox"
	"net/http"
	"time"
)

type config struct {
	resolver Resolver
	resp     fox.HandlerFunc
	filters  []Filter
}

type Option interface {
	apply(*config)
}

type Filter func(c fox.Context) (skip bool)

// Resolver defines the interface for resolving a timeout duration dynamically based on [fox.Context].
// A [time.Duration] is returned if a custom timeout is applicable, along with a boolean indicating if the
// duration was resolved.
type Resolver interface {
	Resolve(c fox.Context) (dt time.Duration, ok bool)
}

// The TimeoutResolverFunc type is an adapter to allow the use of ordinary functions as [Resolver]. If f is a
// function with the appropriate signature, TimeoutResolverFunc(f) is a TimeoutResolverFunc that calls f.
type TimeoutResolverFunc func(c fox.Context) (dt time.Duration, ok bool)

// Resolve calls f(c).
func (f TimeoutResolverFunc) Resolve(c fox.Context) (dt time.Duration, ok bool) {
	return f(c)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func defaultConfig() *config {
	return &config{
		resp: DefaultTimeoutResponse,
	}
}

// WithFilter appends the provided filters to the middleware's filter list.
// A filter returning true will exclude the request from using the timeout handler. If no filters
// are provided, all requests will be handled. Keep in mind that filters are invoked for each request,
// so they should be simple and efficient.
func WithFilter(f ...Filter) Option {
	return optionFunc(func(c *config) {
		c.filters = f
	})
}

// WithResponse sets a custom response handler function for the middleware.
// This function will be invoked when a timeout occurs, allowing for custom responses
// to be sent back to the client. If not set, the middleware use [DefaultTimeoutResponse].
func WithResponse(h fox.HandlerFunc) Option {
	return optionFunc(func(c *config) {
		if h != nil {
			c.resp = h
		}
	})
}

// DefaultTimeoutResponse sends a default 503 Service Unavailable response.
func DefaultTimeoutResponse(c fox.Context) {
	http.Error(c.Writer(), http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
}

// WithTimeoutResolver sets a custom [Resolver] to determine the timeout dynamically based on [fox.Context].
// If the resolver returns false, the default timeout is applied. Keep in mind that a resolver is invoked for each request,
// so they should be simple and efficient.
func WithTimeoutResolver(resolver Resolver) Option {
	return optionFunc(func(c *config) {
		c.resolver = resolver
	})
}
