// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxtimeout/blob/master/LICENSE.txt.
//
// This package is based on the Go standard library, see the LICENSE file
// at https://github.com/golang/go/blob/master/LICENSE.

package foxtimeout

import (
	"bytes"
	"context"
	"fmt"
	"github.com/tigerwill90/fox"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"
)

var bufp = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(nil)
	},
}

// Timeout is a middleware that ensure HTTP handlers don't exceed the configured timeout duration.
type Timeout struct {
	cfg *config
	dt  time.Duration
}

// Middleware returns a fox.MiddlewareFunc with a specified timeout and options.
// This middleware function, when used, will ensure HTTP handlers don't exceed the given timeout duration.
func Middleware(dt time.Duration, opts ...Option) fox.MiddlewareFunc {
	return New(dt, opts...).Timeout
}

// New creates and initializes a new Timeout middleware with the given timeout duration
// and optional settings.3
func New(dt time.Duration, opts ...Option) *Timeout {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	return &Timeout{
		dt:  dt,
		cfg: cfg,
	}
}

// Timeout returns a fox.HandlerFunc that runs next with the given time limit.
//
// The new handler calls next to handle each request, but if a call runs for longer than its time limit,
// the handler responds with a 503 Service Unavailable error and the given message in its body (if a custom response
// handler is not configured). After such a timeout, writes by next to its ResponseWriter will return http.ErrHandlerTimeout.
//
// Timeout supports the Pusher interface but does not support the Hijacker or Flusher interfaces.
func (t *Timeout) Timeout(next fox.HandlerFunc) fox.HandlerFunc {
	if t.dt <= 0 {
		return func(c fox.Context) {
			next(c)
		}
	}

	return func(c fox.Context) {

		ctx, cancel := context.WithTimeout(c.Ctx(), t.dt)
		defer cancel()

		for _, f := range t.cfg.filters {
			if !f(c.Request()) {
				next(c)
				return
			}
		}

		req := c.Request().WithContext(ctx)
		done := make(chan struct{})
		panicChan := make(chan any, 1)

		w := c.Writer()
		buf := bufp.Get().(*bytes.Buffer)
		buf.Reset()
		tw := &timeoutWriter{
			w:       w,
			headers: make(http.Header),
			req:     req,
			code:    http.StatusOK,
			buf:     buf,
		}

		cp := c.CloneWith(tw, req)

		go func() {
			defer func() {
				cp.Close()
				if p := recover(); p != nil {
					panicChan <- p
				}
			}()
			next(cp)
			close(done)
		}()

		select {
		case p := <-panicChan:
			// Don't forget to release the buffer
			bufp.Put(buf)
			panic(p)
		case <-done:
			tw.mu.Lock()
			defer tw.mu.Unlock()
			dst := w.Header()
			for k, vv := range tw.headers {
				dst[k] = vv
			}
			w.WriteHeader(tw.code)
			_, _ = w.Write(tw.buf.Bytes())
		case <-ctx.Done():
			tw.mu.Lock()
			defer tw.mu.Unlock()
			switch err := ctx.Err(); err {
			case context.DeadlineExceeded:
				tw.err = http.ErrHandlerTimeout
			default:
				tw.err = err
			}
			t.cfg.resp(c)
		}
		// Don't forget to release the buffer
		bufp.Put(buf)
	}
}

func checkWriteHeaderCode(code int) {
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid status code %d", code))
	}
}

func relevantCaller() runtime.Frame {
	pc := make([]uintptr, 16)
	n := runtime.Callers(1, pc)
	frames := runtime.CallersFrames(pc[:n])
	var frame runtime.Frame
	for {
		f, more := frames.Next()
		if !strings.HasPrefix(f.Function, "github.com/tigerwill90/foxtimeout.") {
			return f
		}
		if !more {
			break
		}
	}
	return frame
}
