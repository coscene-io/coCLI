package cmd_utils

import (
	"net"
	"net/http"
	"time"

	"github.com/coscene-io/cocli/internal/tlsconfig"
)

func NewTransport(timeout time.Duration) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   16,
		ResponseHeaderTimeout: timeout,
		IdleConnTimeout:       time.Minute,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 10 * time.Second,
		// Set this value so that the underlying transport round-tripper
		// doesn't try to auto decode the body of objects with
		// content-encoding set to `gzip`.
		//
		// Refer:
		//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
		DisableCompression: true,
		TLSClientConfig:    tlsconfig.NewGeneral(),
	}

}

// NewHTTPClient returns an HTTP client using the general outbound TLS policy.
func NewHTTPClient() *http.Client {
	return &http.Client{Transport: NewTransport(0)}
}
