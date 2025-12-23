package probe

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// Measurement represents a single network measurement result
type Measurement struct {
	Target         string
	DNSMs          float64
	TCPMs          float64
	TLSMs          float64
	HTTPTTFBMs     float64
	ThroughputKbps float64
	ErrorStage     *string
	Timestamp      time.Time
}

// MeasurementError represents an error at a specific stage
type MeasurementError struct {
	Stage   string
	Message string
}

func (e *MeasurementError) Error() string {
	return fmt.Sprintf("%s error: %s", e.Stage, e.Message)
}

// MeasureTarget performs a complete network measurement for a target URL
//
// Requirements: 1.1, 1.2, 1.3, 1.4, 1.5
func MeasureTarget(targetURL string) (*Measurement, error) {
	measurement := &Measurement{
		Target:    targetURL,
		Timestamp: time.Now(),
	}

	// Parse URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		errorStage := "parse"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "parse", Message: err.Error()}
	}

	// Ensure scheme is present
	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "https"
	}

	host := parsedURL.Host
	if parsedURL.Port() == "" {
		if parsedURL.Scheme == "https" {
			host = net.JoinHostPort(parsedURL.Hostname(), "443")
		} else {
			host = net.JoinHostPort(parsedURL.Hostname(), "80")
		}
	}

	// Measure DNS resolution time
	dnsStart := time.Now()
	ips, err := net.LookupIP(parsedURL.Hostname())
	measurement.DNSMs = float64(time.Since(dnsStart).Microseconds()) / 1000.0
	if err != nil {
		errorStage := "DNS"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "DNS", Message: err.Error()}
	}
	if len(ips) == 0 {
		errorStage := "DNS"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "DNS", Message: "no IP addresses found"}
	}

	// Measure TCP connection time
	tcpStart := time.Now()
	conn, err := net.DialTimeout("tcp", host, 10*time.Second)
	measurement.TCPMs = float64(time.Since(tcpStart).Microseconds()) / 1000.0
	if err != nil {
		errorStage := "TCP"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "TCP", Message: err.Error()}
	}
	defer conn.Close()

	// Measure TLS handshake time (if HTTPS)
	var httpConn net.Conn = conn
	if parsedURL.Scheme == "https" {
		tlsStart := time.Now()
		tlsConfig := &tls.Config{
			ServerName:         parsedURL.Hostname(),
			InsecureSkipVerify: false,
		}
		tlsConn := tls.Client(conn, tlsConfig)
		err = tlsConn.Handshake()
		measurement.TLSMs = float64(time.Since(tlsStart).Microseconds()) / 1000.0
		if err != nil {
			errorStage := "TLS"
			measurement.ErrorStage = &errorStage
			return measurement, &MeasurementError{Stage: "TLS", Message: err.Error()}
		}
		httpConn = tlsConn
	} else {
		measurement.TLSMs = 0
	}

	// Measure HTTP TTFB (time to first byte)
	// Create HTTP request
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		errorStage := "HTTP"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "HTTP", Message: err.Error()}
	}

	// Send request and measure TTFB
	httpStart := time.Now()

	// Create custom transport to use our existing connection
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return httpConn, nil
			},
			DisableKeepAlives: true,
		},
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	measurement.HTTPTTFBMs = float64(time.Since(httpStart).Microseconds()) / 1000.0
	if err != nil {
		errorStage := "HTTP"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "HTTP", Message: err.Error()}
	}
	defer resp.Body.Close()

	// Read response body to completion (needed for accurate timing)
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		errorStage := "HTTP"
		measurement.ErrorStage = &errorStage
		return measurement, &MeasurementError{Stage: "HTTP", Message: err.Error()}
	}

	return measurement, nil
}

// MeasureThroughput measures download throughput by downloading a 1MB file
//
// Requirement: 4.3 - Download 1MB fixed-size objects over HTTPS with fresh connections
func MeasureThroughput(targetURL string) (float64, error) {
	// Create HTTP client with no keep-alive to force fresh connections
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives:   true,
			DisableCompression:  true,
			MaxIdleConns:        0,
			MaxIdleConnsPerHost: 0,
		},
	}

	// Create request with cache-busting headers
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add cache-busting headers to force fresh download
	req.Header.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Expires", "0")

	// Perform request and measure time
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to perform request: %w", err)
	}
	defer resp.Body.Close()

	// Read all data
	bytesRead, err := io.Copy(io.Discard, resp.Body)
	duration := time.Since(start)

	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	// Calculate throughput in kilobits per second
	// bytesRead * 8 (bits) / duration (seconds) / 1000 (kbps)
	throughputKbps := float64(bytesRead*8) / duration.Seconds() / 1000.0

	return throughputKbps, nil
}

// MeasureTargetWithThroughput performs a complete measurement including throughput
func MeasureTargetWithThroughput(baseURL, throughputURL string) (*Measurement, error) {
	// First measure timing
	measurement, err := MeasureTarget(baseURL)
	if err != nil {
		return measurement, err
	}

	// Then measure throughput separately
	throughput, err := MeasureThroughput(throughputURL)
	if err != nil {
		// Set error stage but don't fail the entire measurement
		errorStage := "throughput"
		measurement.ErrorStage = &errorStage
		measurement.ThroughputKbps = 0
		return measurement, &MeasurementError{Stage: "throughput", Message: err.Error()}
	}

	measurement.ThroughputKbps = throughput
	return measurement, nil
}
