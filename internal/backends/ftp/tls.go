// SPDX-License-Identifier: MIT

package ftp

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"github.com/ebogdum/filemanager/internal/plugin"
)

// buildTLSConfig builds a TLS configuration from the backend config.
func buildTLSConfig(config plugin.BackendConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.TLSSkipVerify,
		MinVersion:         tls.VersionTLS12,
	}

	// Load CA certificate if provided
	if len(config.TLSCA) > 0 {
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(config.TLSCA) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = certPool
	}

	// Load client certificate if provided
	if len(config.TLSCert) > 0 && len(config.TLSKey) > 0 {
		cert, err := tls.X509KeyPair(config.TLSCert, config.TLSKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Set server name for SNI if host is provided
	if config.Host != "" {
		tlsConfig.ServerName = config.Host
	}

	return tlsConfig, nil
}
