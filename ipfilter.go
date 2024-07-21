// Copyright 2024 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxgeoip/blob/master/LICENSE.

package foxgeoip

import (
	"context"
	"github.com/oschwald/geoip2-golang"
	"github.com/tigerwill90/fox"
	"log/slog"
	"net"
	"net/http"
	"strings"
)

type IPFilter struct {
	strategy     fox.ClientIPStrategy
	r            *geoip2.Reader
	cfg          *config
	blockHandler fox.HandlerFunc
	countryCodes countryCodes
	logger       *slog.Logger
	isWhitelist  bool
}

// New creates a new IPFilter with the provided GeoIP2 reader and options. The ip filter is intended to work with
// MaxMind GeoLite2 or GeoIP2 databases. It should work with other MMDB databases but has not been tested.
// Note that blacklist and whitelist options are mutually exclusive. Either it is a whitelist, and all requests are
// denied except for IPs that have a country code associated in the whitelist, OR it is a blacklist, and all requests are
// allowed except IPs that have a country code associated in the blacklist.
func New(db *geoip2.Reader, opts ...Option) *IPFilter {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt.apply(cfg)
	}

	f := &IPFilter{
		r:            db,
		cfg:          cfg,
		strategy:     cfg.strategy,
		blockHandler: cfg.blockHandler,
		logger:       slog.New(cfg.handler),
	}

	whitelist := normalizeCodes(cfg.whitelist)
	if len(whitelist) > 0 {
		f.isWhitelist = true
		f.countryCodes = whitelist
		return f
	}

	f.countryCodes = normalizeCodes(cfg.blacklist)
	return f
}

// Middleware creates a middleware function for the IP filter. The middleware is intended to work with
// MaxMind GeoLite2 or GeoIP2 databases. It should work with other MMDB databases but has not been tested.
// Note that blacklist and whitelist options are mutually exclusive. Either it is a whitelist, and all requests are
// denied except for IPs that have a country code associated in the whitelist, OR it is a blacklist, and all requests are
// allowed except IPs that have a country code associated in the blacklist.
func Middleware(db *geoip2.Reader, opts ...Option) fox.MiddlewareFunc {
	return New(db, opts...).FilterIP
}

// FilterIP is a middleware function that filters requests based on the IP address.
func (f *IPFilter) FilterIP(next fox.HandlerFunc) fox.HandlerFunc {
	return func(c fox.Context) {

		ctx := c.Request().Context()

		for _, filter := range f.cfg.filters {
			if filter(c.Request()) {
				f.logger.DebugContext(ctx, "geoip: skipping request due to filter match")
				next(c)
				return
			}
		}

		var ipAddr *net.IPAddr
		var err error
		if f.strategy == nil {
			ipAddr, err = c.ClientIP()
		} else {
			ipAddr, err = f.strategy.ClientIP(c)
		}

		if err != nil {
			f.logger.ErrorContext(ctx, "geoip: failed to derive client ip", slog.String("error", err.Error()))
			http.Error(c.Writer(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		allowed, code, err := f.Allowed(ipAddr.IP)
		if err != nil {
			f.logger.ErrorContext(
				ctx,
				"geoip: unexpected lookup error",
				slog.String("ip", ipAddr.String()),
				slog.String("country", code),
				slog.String("error", err.Error()),
			)
			http.Error(c.Writer(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if !allowed {
			f.logger.WarnContext(
				ctx,
				"geoip: blocking ip address",
				slog.String("ip", ipAddr.String()),
				slog.String("country", code),
			)
			f.blockHandler(c)
			return
		}

		next(c)
	}
}

// DefaultBlockingResponse is the default response for blocked IPs.
// It responds with a 403 Forbidden http status.
func DefaultBlockingResponse(c fox.Context) {
	c.Writer().WriteHeader(http.StatusForbidden)
}

// Allowed checks if the given IP address is allowed based on the filter's configuration.
func (f *IPFilter) Allowed(ip net.IP) (allowed bool, code string, err error) {
	allowed, code, err = f.allowed(f.countryCodes, ip)
	if err != nil {
		return
	}
	return allowed == f.isWhitelist, code, nil
}

func (f *IPFilter) allowed(codes countryCodes, ip net.IP) (allowed bool, code string, err error) {
	country, err := f.r.Country(ip)
	if err != nil {
		return false, "", err
	}

	code = country.Country.IsoCode
	// Default to not in the list
	if len(code) == 0 {
		return
	}

	return codes.has(code), code, nil
}

type countryCodes map[string]struct{}

func (c countryCodes) has(code string) bool {
	_, ok := c[code]
	return ok
}

// normalizeCodes standardizes country codes to uppercase, removes empty entries, and ensures uniqueness.
// It returns a countryCodes map with the processed codes.
func normalizeCodes(codes []string) countryCodes {
	if len(codes) == 0 {
		return nil
	}

	normalizedCodes := make(map[string]struct{})
	for _, code := range codes {
		if len(code) == 0 {
			continue
		}
		capCode := strings.ToUpper(strings.TrimSpace(code))
		if _, ok := normalizedCodes[capCode]; !ok {
			normalizedCodes[capCode] = struct{}{}
		}
	}
	return normalizedCodes
}

var _ slog.Handler = (*noopHandler)(nil)

type noopHandler struct {
	level slog.Level
}

func (n noopHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= n.level
}

func (n noopHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (n noopHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return noopHandler{}
}

func (n noopHandler) WithGroup(_ string) slog.Handler {
	return noopHandler{}
}
