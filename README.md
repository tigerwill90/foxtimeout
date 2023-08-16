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

### Usage
````go
package main

import (
	"github.com/tigerwill90/fox"
	"github.com/tigerwill90/foxtimeout"
	"log"
	"net/http"
	"time"
)

func main() {

	f := fox.New(
		fox.DefaultOptions(),
		fox.WithMiddlewareFor(fox.RouteHandlers, foxtimeout.Middleware(50*time.Microsecond)),
	)
	f.MustHandle(http.MethodGet, "/hello/{name}", func(c fox.Context) {
		time.Sleep(10 * time.Millisecond)
		_ = c.String(http.StatusOK, "hello %s\n", c.Param("name"))
	})

	log.Fatalln(http.ListenAndServe(":8080", f))
}
````