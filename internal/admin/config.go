package admin

// Config holds admin service configuration
type Config struct {
	TLS TLSConfig
}

// TLSConfig holds TLS/HTTPS configuration
type TLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}
