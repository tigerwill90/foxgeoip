package foxgeoip

import (
	"github.com/tigerwill90/fox"
	"log/slog"
	"net/http"
)

type config struct {
	blacklist    []string
	whitelist    []string
	filters      []Filter
	strategy     fox.ClientIPStrategy
	handler      slog.Handler
	blockHandler fox.HandlerFunc
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

type Filter func(r *http.Request) bool

func WithBlacklistedCountries(codes ...string) Option {
	return optionFunc(func(c *config) {
		c.whitelist = nil
		c.blacklist = append(c.blacklist, codes...)
	})
}

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

func WithClientIPStrategy(strategy fox.ClientIPStrategy) Option {
	return optionFunc(func(c *config) {
		if strategy != nil {
			c.strategy = strategy
		}
	})
}

func WithLogHandler(handler slog.Handler) Option {
	return optionFunc(func(c *config) {
		if handler != nil {
			c.handler = handler
		}
	})
}

func WithResponse(handler fox.HandlerFunc) Option {
	return optionFunc(func(c *config) {
		if handler != nil {
			c.blockHandler = handler
		}
	})
}
