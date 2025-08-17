// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxtimeout/blob/master/LICENSE.txt.

package foxtimeout

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tigerwill90/fox"
)

func success201response(c fox.Context) {
	time.Sleep(10 * time.Millisecond)
	_ = c.String(http.StatusCreated, "%s\n", http.StatusText(http.StatusCreated))
}

func TestMiddleware_WithTimeout(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(50 * time.Microsecond)))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusServiceUnavailable)), w.Body.String())
}

func TestMiddleware_WithoutTimeout(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(1 * time.Second)))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func timeoutResponse(c fox.Context) {
	http.Error(c.Writer(), http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
}

func TestMiddleware_WithResponse(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(50*time.Microsecond, WithResponse(timeoutResponse))))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestTimeout, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusRequestTimeout)), w.Body.String())
}

func panicResponse(c fox.Context) {
	panic("test")
}

func TestMiddleware_WithPanic(t *testing.T) {
	f, err := fox.New(
		fox.WithMiddleware(
			fox.CustomRecoveryWithLogHandler(slog.NewTextHandler(io.Discard, nil), func(c fox.Context, err any) {
				if !c.Writer().Written() {
					http.Error(c.Writer(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}),
			Middleware(1*time.Second, WithResponse(timeoutResponse)),
		),
	)
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", panicResponse)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusInternalServerError)), w.Body.String())
}

func TestMiddleware_NoTimeout(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(0)))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", success201response)

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func TestMiddleware_ErrNotSupported(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(1 * time.Second)))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", func(c fox.Context) {
		assert.ErrorIs(t, c.Writer().FlushError(), http.ErrNotSupported)
		_, _, hijErr := c.Writer().Hijack()
		assert.ErrorIs(t, hijErr, http.ErrNotSupported)
		assert.ErrorIs(t, c.Writer().SetReadDeadline(time.Now()), http.ErrNotSupported)
		assert.ErrorIs(t, c.Writer().SetWriteDeadline(time.Now()), http.ErrNotSupported)
	})

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)
}

func TestMiddleware_WithAfter(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(1 * time.Millisecond)))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", success201response, After(2*time.Second))

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func TestMiddleware_WithNone(t *testing.T) {
	f, err := fox.New(fox.WithMiddleware(Middleware(1 * time.Millisecond)))
	require.NoError(t, err)
	f.MustHandle(http.MethodGet, "/foo", success201response, None())

	req := httptest.NewRequest(http.MethodGet, "/foo", nil)
	w := httptest.NewRecorder()
	f.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, fmt.Sprintf("%s\n", http.StatusText(http.StatusCreated)), w.Body.String())
}

func ExampleAfter() {
	f, err := fox.New(
		fox.WithMiddleware(Middleware(2 * time.Second)),
	)
	if err != nil {
		panic(err)
	}

	f.MustHandle(http.MethodGet, "/hello/{name}", func(c fox.Context) {
		_ = c.String(http.StatusOK, "hello %s\n", c.Param("name"))
	})
	f.MustHandle(http.MethodGet, "/long", func(c fox.Context) {
		time.Sleep(10 * time.Second)
		c.Writer().WriteHeader(http.StatusOK)
	}, After(12*time.Second))
}

func ExampleNone() {
	f, err := fox.New(
		fox.WithMiddleware(Middleware(2 * time.Second)),
	)
	if err != nil {
		panic(err)
	}

	f.MustHandle(http.MethodGet, "/hello/{name}", func(c fox.Context) {
		_ = c.String(http.StatusOK, "hello %s\n", c.Param("name"))
	})
	f.MustHandle(http.MethodGet, "/long", func(c fox.Context) {
		time.Sleep(10 * time.Second)
		c.Writer().WriteHeader(http.StatusOK)
	}, None())
}
