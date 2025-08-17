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
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/tigerwill90/fox"
)

var (
	bufp = sync.Pool{
		New: func() any {
			return bytes.NewBuffer(nil)
		},
	}
)

// Timeout is a middleware that ensure HTTP handlers don't exceed the configured timeout duration.
type Timeout struct {
	cfg *config
	dt  time.Duration
}

// Middleware returns a [fox.MiddlewareFunc] that runs handlers with the given time limit.
//
// The middleware calls the next handler to handle each request, but if a call runs for longer than its time limit,
// the handler responds with a 503 Service Unavailable error and the given message in its body (if a custom response
// handler is not configured). After such a timeout, writes by the handler to its ResponseWriter will return [http.ErrHandlerTimeout].
//
// The timeout middleware supports the [http.Pusher] interface but does not support the [http.Hijacker] or [http.Flusher] interfaces.
//
// Individual routes can override the timeout duration using the [After] option or disable it entirely using [None]:
func Middleware(dt time.Duration, opts ...Option) fox.MiddlewareFunc {
	return create(dt, opts...).run
}

func create(dt time.Duration, opts ...Option) *Timeout {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	return &Timeout{
		dt:  dt,
		cfg: cfg,
	}
}

// run is the internal handler that applies the timeout logic.
func (t *Timeout) run(next fox.HandlerFunc) fox.HandlerFunc {
	return func(c fox.Context) {

		for _, f := range t.cfg.filters {
			if f(c) {
				next(c)
				return
			}
		}

		dt := t.resolveTimeout(c)
		if dt <= 0 {
			next(c)
			return
		}

		ctx, cancel := context.WithTimeout(c.Request().Context(), dt)
		defer cancel()

		req := c.Request().WithContext(ctx)
		done := make(chan struct{})
		panicChan := make(chan any, 1)

		w := c.Writer()
		buf := bufp.Get().(*bytes.Buffer)
		defer bufp.Put(buf)
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
			if t.cfg.enableAbortRequestBody {
				_ = w.SetReadDeadline(time.Now())
			}
			t.cfg.resp(c)
		}
	}
}

func (t *Timeout) resolveTimeout(c fox.Context) time.Duration {
	if dt, ok := unwrapRouteTimeout(c.Route()); ok {
		return dt
	}
	return t.dt
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
