// Copyright 2023 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxtimeout/blob/master/LICENSE.txt.
//
// This package is based on the Go standard library, see the LICENSE file
// at https://github.com/golang/go/blob/master/LICENSE.

package foxtimeout

import (
	"bufio"
	"bytes"
	"github.com/tigerwill90/fox"
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"sync"
	"time"
)

var _ fox.ResponseWriter = (*timeoutWriter)(nil)

var copyBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 32*1024)
		return &b
	},
}

type onlyWrite struct {
	io.Writer
}

type timeoutWriter struct {
	w       fox.ResponseWriter
	err     error
	headers http.Header
	req     *http.Request
	buf     *bytes.Buffer
	code    int
	mu      sync.RWMutex
	written bool
	n       int
}

func (tw *timeoutWriter) Status() int {
	return tw.code
}

func (tw *timeoutWriter) Written() bool {
	return tw.written
}

func (tw *timeoutWriter) Size() int {
	return tw.n
}

func (tw *timeoutWriter) WriteString(s string) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.err != nil {
		return 0, tw.err
	}
	if !tw.written {
		tw.writeHeaderLocked(http.StatusOK)
	}

	n, err := io.WriteString(tw.buf, s)
	tw.n += n
	return n, err
}

func (tw *timeoutWriter) Push(target string, opts *http.PushOptions) error {
	return tw.w.Push(target, opts)
}

func (tw *timeoutWriter) Header() http.Header {
	return tw.headers
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.err != nil {
		return 0, tw.err
	}
	if !tw.written {
		tw.writeHeaderLocked(http.StatusOK)
	}

	n, err := tw.buf.Write(p)
	tw.n += n
	return n, err
}

func (tw *timeoutWriter) writeHeaderLocked(code int) {
	checkWriteHeaderCode(code)
	switch {
	case tw.err != nil:
		return
	case tw.written:
		caller := relevantCaller()
		log.Printf("http: superfluous response.WriteHeader call from %s (%s:%d)", caller.Function, path.Base(caller.File), caller.Line)
	default:
		tw.written = true
		tw.code = code
	}
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.writeHeaderLocked(code)
}

func (tw *timeoutWriter) ReadFrom(src io.Reader) (n int64, err error) {
	bufPtr := copyBufPool.Get().(*[]byte)
	buf := *bufPtr
	// onlyWrite hide "ReadFrom" from w.
	n, err = io.CopyBuffer(onlyWrite{tw}, src, buf)
	copyBufPool.Put(bufPtr)
	return
}

func (tw *timeoutWriter) FlushError() error {
	return fox.ErrNotSupported()
}

func (tw *timeoutWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, fox.ErrNotSupported()
}

func (tw *timeoutWriter) SetReadDeadline(deadline time.Time) error {
	return fox.ErrNotSupported()
}

func (tw *timeoutWriter) SetWriteDeadline(deadline time.Time) error {
	return fox.ErrNotSupported()
}
