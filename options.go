// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxtimeout/blob/master/LICENSE.txt.

package foxtimeout

import (
	"github.com/tigerwill90/fox"
	"net/http"
)

type config struct {
	resp    fox.HandlerFunc
	filters []Filter
}

type Option interface {
	apply(*config)
}

type Filter func(r *http.Request) bool

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
// A filter returning false will exclude the request from using the timeout handler. If no filters
// are provided, all requests will be handled. Keep in mind that filters are invoked for each request,
// so they should be simple and efficient.
func WithFilter(f ...Filter) Option {
	return optionFunc(func(c *config) {
		c.filters = f
	})
}

// WithResponse sets a custom response handler function for the middleware.
// This function will be invoked when a timeout occurs, allowing for custom responses
// to be sent back to the client. If not set, the middleware use DefaultTimeoutResponse.
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
