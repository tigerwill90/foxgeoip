// Copyright 2024 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxgeoip/blob/master/LICENSE.

package foxgeoip

import (
	"github.com/tigerwill90/fox"
	"log/slog"
)

type config struct {
	strategy     fox.ClientIPStrategy
	handler      slog.Handler
	blockHandler fox.HandlerFunc
	blacklist    []string
	whitelist    []string
	filters      []Filter
}

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(c *config) {
	f(c)
}

func defaultConfig() *config {
	return &config{
		blockHandler: DefaultBlockingResponse,
		handler:      noopHandler{slog.LevelDebug},
	}
}

type Filter func(c fox.Context) bool

// WithBlacklistedCountries sets the blacklist with the provided country codes.
// It clears any existing whitelist. Requests from countries in the blacklist will be denied.
func WithBlacklistedCountries(codes ...string) Option {
	return optionFunc(func(c *config) {
		c.whitelist = nil
		c.blacklist = append(c.blacklist, codes...)
	})
}

// WithWhitelistedCountries sets the whitelist with the provided country codes.
// It clears any existing blacklist. Requests from countries not in the whitelist will be denied.
func WithWhitelistedCountries(codes ...string) Option {
	return optionFunc(func(c *config) {
		c.blacklist = nil
		c.whitelist = append(c.whitelist, codes...)
	})
}

// WithFilter appends the provided filters to the middleware's filter list.
// A filter returning true will exclude the request from using the ip filter handler. If no filters
// are provided, all requests will be handled. Keep in mind that filters are invoked for each request,
// so they should be simple and efficient.
func WithFilter(f ...Filter) Option {
	return optionFunc(func(c *config) {
		c.filters = append(c.filters, f...)
	})
}

// WithClientIPStrategy sets a custom strategy to determine the client IP address.
// This is for advanced use case, you should configure the strategy with Fox's router option using
// fox.WithClientIPStrategy.
func WithClientIPStrategy(strategy fox.ClientIPStrategy) Option {
	return optionFunc(func(c *config) {
		if strategy != nil {
			c.strategy = strategy
		}
	})
}

// WithLogHandler sets a custom log handler for structured logging.
func WithLogHandler(handler slog.Handler) Option {
	return optionFunc(func(c *config) {
		if handler != nil {
			c.handler = handler
		}
	})
}

// WithResponse sets a custom response handler for blocked requests.
func WithResponse(handler fox.HandlerFunc) Option {
	return optionFunc(func(c *config) {
		if handler != nil {
			c.blockHandler = handler
		}
	})
}
