package transport

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
)

type ClientOptions struct {
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	IdleConnTimeout     time.Duration
	ConnectTimeout      time.Duration
	KeepAlive           time.Duration
	ResponseTimeout     time.Duration
	TLSHandshakeTimeout time.Duration
	InsecureSkipVerify  bool
	CustomCACertPath    string
}

func DefaultClientOptions() ClientOptions {
	return ClientOptions{
		MaxIdleConns:        512,
		MaxIdleConnsPerHost: 256,
		MaxConnsPerHost:     512,
		IdleConnTimeout:     90 * time.Second,
		ConnectTimeout:      10 * time.Second,
		KeepAlive:           30 * time.Second,
		ResponseTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		InsecureSkipVerify:  false,
		CustomCACertPath:    "",
	}
}

func NewHTTPClient(opts ClientOptions) (*http.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: opts.InsecureSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	if opts.CustomCACertPath != "" {
		caCert, err := os.ReadFile(opts.CustomCACertPath)
		if err != nil {
			return nil, fmt.Errorf("read custom CA cert: %w", err)
		}
		caCertPool, err := x509.SystemCertPool()
		if err != nil || caCertPool == nil {
			caCertPool = x509.NewCertPool()
		}
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append custom CA cert from %s", opts.CustomCACertPath)
		}
		tlsConfig.RootCAs = caCertPool
	}

	dialer := &net.Dialer{
		Timeout:   opts.ConnectTimeout,
		KeepAlive: opts.KeepAlive,
	}

	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		MaxIdleConns:          opts.MaxIdleConns,
		MaxIdleConnsPerHost:   opts.MaxIdleConnsPerHost,
		MaxConnsPerHost:       opts.MaxConnsPerHost,
		IdleConnTimeout:       opts.IdleConnTimeout,
		TLSHandshakeTimeout:   opts.TLSHandshakeTimeout,
		TLSClientConfig:       tlsConfig,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     true,
	}

	if err := http2.ConfigureTransport(tr); err != nil {
		return nil, fmt.Errorf("configure http2 transport: %w", err)
	}

	return &http.Client{
		Transport: tr,
		Timeout:   opts.ResponseTimeout,
	}, nil
}
