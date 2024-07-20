// Copyright 2024 Sylvain Müller. All rights reserved.
// Mount of this source code is governed by a MIT license that can be found
// at https://github.com/tigerwill90/foxgeoip/blob/master/LICENSE.

package foxgeoip

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tigerwill90/fox"
	"github.com/tigerwill90/fox/strategy"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const (
	egUS = "52.92.180.128"
	egAU = "49.189.50.1"
	egCN = "116.31.116.51"
)

func TestAllowed(t *testing.T) {

	r, err := geoip2.Open("testdata/GeoLite2-Country-outdated.mmdb")
	require.NoError(t, err)

	cases := []struct {
		name     string
		opts     []Option
		ip       net.IP
		want     bool
		wantCode string
	}{
		{
			name:     "no blacklist or whitelist, default to allow all",
			ip:       net.ParseIP(egUS),
			want:     true,
			wantCode: "US",
		},
		{
			name:     "whitelist CH, FR and UK, deny US",
			opts:     []Option{WithWhitelistedCountries("FR", "CH", "UK")},
			ip:       net.ParseIP(egUS),
			want:     false,
			wantCode: "US",
		},
		{
			name: "whitelist CH, FR and UK, ip not in db",
			opts: []Option{WithWhitelistedCountries("FR", "CH", "UK")},
			ip:   net.ParseIP("127.0.0.1"),
			want: false,
		},
		{
			name: "whitelist CH, FR, UK and empty, ip not in db",
			opts: []Option{WithWhitelistedCountries("FR", "CH", "UK", "", "CH")},
			ip:   net.ParseIP("127.0.0.1"),
			want: false,
		},
		{
			name: "whitelist CH, US and UK, ipv6 not in db",
			opts: []Option{WithWhitelistedCountries("CH", "US", "UK")},
			ip:   net.ParseIP("2001:0db8:3c4d:0015:0000:0000:1a2f:1a2b"),
			want: false,
		},
		{
			name:     "whitelist CH, US and UK, allow US",
			opts:     []Option{WithWhitelistedCountries("CH", "US", "UK")},
			ip:       net.ParseIP(egUS),
			want:     true,
			wantCode: "US",
		},
		{
			name:     "blacklist CH, US and UK, deny US",
			opts:     []Option{WithBlacklistedCountries("CH", "US", "UK")},
			ip:       net.ParseIP(egUS),
			want:     false,
			wantCode: "US",
		},
		{
			name:     "blacklist CH, US and UK, allow CN",
			opts:     []Option{WithBlacklistedCountries("CH", "US", "UK")},
			ip:       net.ParseIP(egCN),
			want:     true,
			wantCode: "CN",
		},
		{
			name:     "blacklist ch, us, and au in lowercase, deny AU",
			opts:     []Option{WithBlacklistedCountries("ch", "us", "au")},
			ip:       net.ParseIP(egAU),
			want:     false,
			wantCode: "AU",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ipfilter := New(r, tc.opts...)
			allowed, code, err := ipfilter.Allowed(tc.ip)
			require.NoError(t, err)
			assert.Equal(t, tc.want, allowed)
			assert.Equal(t, tc.wantCode, code)
		})
	}
}

func TestMiddleware(t *testing.T) {
	r, err := geoip2.Open("testdata/GeoLite2-Country-outdated.mmdb")
	require.NoError(t, err)

	cases := []struct {
		name       string
		f          *fox.Router
		remoteAddr string
		wantStatus int
	}{
		{
			name: "no blacklist or whitelist, default to allow all",
			f: fox.New(
				fox.WithClientIPStrategy(
					strategy.NewRemoteAddr(),
				),
				fox.WithMiddleware(
					Middleware(r),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "whitelist CH, FR and UK, deny US",
			f: fox.New(
				fox.WithClientIPStrategy(
					strategy.NewRemoteAddr(),
				),
				fox.WithMiddleware(
					Middleware(
						r,
						WithWhitelistedCountries("FR", "CH", "UK"),
					),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "whitelist AU, FR and UK, allow AU",
			f: fox.New(
				fox.WithClientIPStrategy(
					strategy.NewRemoteAddr(),
				),
				fox.WithMiddleware(
					Middleware(
						r,
						WithWhitelistedCountries("AU", "CH", "UK"),
					),
				),
			),
			remoteAddr: egAU,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "blacklist CH, FR and US, deny US",
			f: fox.New(
				fox.WithClientIPStrategy(
					strategy.NewRemoteAddr(),
				),
				fox.WithMiddleware(
					Middleware(
						r,
						WithBlacklistedCountries("FR", "CH", "US"),
					),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "blacklist CH, FR and US, deny US with custom response",
			f: fox.New(
				fox.WithClientIPStrategy(
					strategy.NewRemoteAddr(),
				),
				fox.WithMiddleware(
					Middleware(
						r,
						WithBlacklistedCountries("FR", "CH", "US"),
						WithResponse(func(c fox.Context) {
							c.Writer().WriteHeader(http.StatusUnauthorized)
						}),
					),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "blacklist CH, FR and US, filter request",
			f: fox.New(
				fox.WithClientIPStrategy(
					strategy.NewRemoteAddr(),
				),
				fox.WithMiddleware(
					Middleware(
						r,
						WithBlacklistedCountries("FR", "CH", "US"),
						WithFilter(func(r *http.Request) bool {
							return true
						}),
					),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusNoContent,
		},
		{
			name: "blacklist CH, FR and US, deny US with custom strategy",
			f: fox.New(
				fox.WithMiddleware(
					Middleware(
						r,
						WithBlacklistedCountries("FR", "CH", "US"),
						WithClientIPStrategy(strategy.NewRemoteAddr()),
					),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusForbidden,
		},
		{
			name: "no ip client strategy",
			f: fox.New(
				fox.WithMiddleware(
					Middleware(
						r,
						WithBlacklistedCountries("FR", "CH", "US"),
					),
				),
			),
			remoteAddr: egUS,
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.f.MustHandle(http.MethodGet, "/foobar", func(c fox.Context) {
				c.Writer().WriteHeader(http.StatusNoContent)
			})
			req := httptest.NewRequest(http.MethodGet, "/foobar", nil)
			req.RemoteAddr = tc.remoteAddr

			w := httptest.NewRecorder()

			tc.f.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)

		})
	}
}
