package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"
)

// TLSConfig holds TLS/HTTPS configuration
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled" json:"enabled"`
	CertFile string `yaml:"cert_file" json:"cert_file"`
	KeyFile  string `yaml:"key_file" json:"key_file"`
	// AutoTLS enables automatic certificate generation using Let's Encrypt
	AutoTLS bool   `yaml:"auto_tls" json:"auto_tls"`
	Domain  string `yaml:"domain" json:"domain"`
	// MinVersion sets minimum TLS version (default: TLS 1.2)
	MinVersion string `yaml:"min_version" json:"min_version"`
}

// Server wraps http.Server with TLS support
type Server struct {
	httpServer *http.Server
	tlsConfig  *TLSConfig
}

// NewServer creates a new server with optional TLS support
func NewServer(addr string, handler http.Handler, tlsConfig *TLSConfig) *Server {
	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Configure TLS if enabled
	if tlsConfig != nil && tlsConfig.Enabled {
		server.TLSConfig = &tls.Config{
			MinVersion:               getTLSVersion(tlsConfig.MinVersion),
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
		}
	}

	return &Server{
		httpServer: server,
		tlsConfig:  tlsConfig,
	}
}

// Start starts the server (with or without TLS)
func (s *Server) Start() error {
	if s.tlsConfig != nil && s.tlsConfig.Enabled {
		if s.tlsConfig.AutoTLS {
			return fmt.Errorf("AutoTLS not yet implemented - use cert_file and key_file for now")
		}

		log.Printf("Starting HTTPS server on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServeTLS(s.tlsConfig.CertFile, s.tlsConfig.KeyFile); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTPS server error: %w", err)
		}
	} else {
		log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// getTLSVersion converts string to tls.Version constant
func getTLSVersion(version string) uint16 {
	switch version {
	case "1.3", "TLS1.3":
		return tls.VersionTLS13
	case "1.2", "TLS1.2":
		return tls.VersionTLS12
	case "1.1", "TLS1.1":
		return tls.VersionTLS11
	default:
		return tls.VersionTLS12 // Default to TLS 1.2
	}
}
