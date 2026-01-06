[![Go Reference](https://pkg.go.dev/badge/github.com/tigerwill90/foxgeoip.svg)](https://pkg.go.dev/github.com/tigerwill90/foxgeoip)
[![tests](https://github.com/tigerwill90/foxgeoip/actions/workflows/tests.yaml/badge.svg)](https://github.com/tigerwill90/foxgeoip/actions?query=workflow%3Atests)
[![Go Report Card](https://goreportcard.com/badge/github.com/tigerwill90/foxgeoip)](https://goreportcard.com/report/github.com/tigerwill90/foxgeoip)
[![codecov](https://codecov.io/github/tigerwill90/foxgeoip/graph/badge.svg?token=KHzVeasvd7)](https://codecov.io/github/tigerwill90/foxgeoip)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tigerwill90/foxgeoip)
![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/tigerwill90/foxgeoip)

# Foxgeoip

FoxGeoIP is an experimental middleware for [Fox](https://github.com/tigerwill90/fox) that filters incoming requests based on the 
client's IP address using GeoIP data. It blocks or allows access based on country codes. This middleware is intended to work with
[MaxMind GeoLite2 or GeoIP2 databases](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data).

## Disclaimer
Foxgeoip's API is closely tied to the Fox router, and it will only reach v1 when the router is stabilized. During the 
pre-v1 phase, breaking changes may occur and will be documented in the release notes.

## Getting started
### Installation

````shell
go get -u github.com/tigerwill90/foxgeoip
````

### Feature
- Filters requests based on country codes, either allowing or blocking them.
- Supports whitelists or blacklists mode.
- Tightly integrates with the Fox ecosystem for enhanced performance and scalability.
- Provides structured logging with `log/slog`.

### Usage
````go
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/oschwald/geoip2-golang"
	"github.com/tigerwill90/fox"
	"github.com/tigerwill90/fox/clientip"
	"github.com/tigerwill90/foxgeoip"
)

func main() {
	db, err := geoip2.Open("GeoLite2-Country.mmdb")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	resolver, err := clientip.NewRightmostNonPrivate(clientip.XForwardedForKey)
	if err != nil {
		panic(err)
	}
	f := fox.MustRouter(
		fox.DefaultOptions(),
		fox.WithClientIPResolver(resolver),
		fox.WithMiddleware(
			foxgeoip.Middleware(
				db,
				foxgeoip.WithBlacklistedCountries("US", "CN", "AU"),
			),
		),
	)

	f.MustAdd(fox.MethodGet, "/hello/{name}", func(c *fox.Context) {
		_ = c.String(http.StatusOK, fmt.Sprintf("hello %s\n", c.Param("name")))
	})

	if err = http.ListenAndServe(":8080", f); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
	}
}
````
