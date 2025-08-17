[![Go Reference](https://pkg.go.dev/badge/github.com/tigerwill90/foxtimeout.svg)](https://pkg.go.dev/github.com/tigerwill90/foxtimeout)
[![tests](https://github.com/tigerwill90/foxtimeout/actions/workflows/tests.yaml/badge.svg)](https://github.com/tigerwill90/foxtimeout/actions?query=workflow%3Atests)
[![Go Report Card](https://goreportcard.com/badge/github.com/tigerwill90/foxtimeout)](https://goreportcard.com/report/github.com/tigerwill90/foxtimeout)
[![codecov](https://codecov.io/gh/tigerwill90/foxtimeout/branch/master/graph/badge.svg?token=D6qSTlzEcE)](https://codecov.io/gh/tigerwill90/foxtimeout)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tigerwill90/foxtimeout)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tigerwill90/foxtimeout)

# Foxtimeout

Foxtimeout is a middleware for [Fox](https://github.com/tigerwill90/fox) which ensure that a handler do not exceed the
configured timeout limit.

## Disclaimer
Foxtimeout's API is closely tied to the Fox router, and it will only reach v1 when the router is stabilized.
During the pre-v1 phase, breaking changes may occur and will be documented in the release notes.

## Getting started
### Installation

````shell
go get -u github.com/tigerwill90/foxtimeout
````
### Feature
- Allows for custom timeout response to better suit specific use cases.
- Tightly integrates with the Fox ecosystem for enhanced performance and scalability.
- Supports dynamic timeout configuration on a per-route & per-request basis using custom `Resolver`.

### Usage
````go
package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/tigerwill90/fox"
	"github.com/tigerwill90/foxtimeout"
)

func main() {
	f, err := fox.New(
		fox.DefaultOptions(),
		fox.WithMiddleware(
			foxtimeout.Middleware(2*time.Second),
		),
	)
	if err != nil {
		panic(err)
	}

	f.MustHandle(http.MethodGet, "/hello/{name}", func(c fox.Context) {
		_ = c.String(http.StatusOK, "hello %s\n", c.Param("name"))
	})
	f.MustHandle(http.MethodGet, "/download/{filepath}", DownloadHandler, foxtimeout.None())
	f.MustHandle(http.MethodGet, "/workflow/{id}/start", WorkflowHandler, foxtimeout.After(15*time.Second))

	if err = http.ListenAndServe(":8080", f); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
	}
}
````
