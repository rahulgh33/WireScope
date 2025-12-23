#!/usr/bin/env python3
"""
Simple test target server for Network QoE testing
Provides endpoints for health checks, configurable delays, and throughput testing
"""

import http.server
import socketserver
import urllib.parse
import time
import os

class TestHandler(http.server.BaseHTTPRequestHandler):
    def do_HEAD(self):
        # Handle HEAD requests by calling GET logic but not sending body
        parsed_path = urllib.parse.urlparse(self.path)
        path = parsed_path.path
        
        if path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'text/plain')
            self.send_header('Cache-Control', 'no-cache')
            self.end_headers()
            return
        
        if path == '/fixed/1mb.bin':
            self.send_response(200)
            self.send_header('Content-Type', 'application/octet-stream')
            self.send_header('Content-Length', '1048576')
            self.send_header('Cache-Control', 'no-cache, no-store, must-revalidate')
            self.send_header('Pragma', 'no-cache')
            self.send_header('Expires', '0')
            self.end_headers()
            return
        
        # Default 404
        self.send_response(404)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()

    def do_GET(self):
        parsed_path = urllib.parse.urlparse(self.path)
        path = parsed_path.path
        query = urllib.parse.parse_qs(parsed_path.query)
        
        # Health endpoint - fast response
        if path == '/health':
            self.send_response(200)
            self.send_header('Content-Type', 'text/plain')
            self.send_header('Cache-Control', 'no-cache')
            self.end_headers()
            self.wfile.write(b'OK')
            return
        
        # Slow endpoint with configurable delay
        if path == '/slow':
            delay_ms = int(query.get('ms', ['1000'])[0])
            delay_seconds = delay_ms / 1000.0
            time.sleep(delay_seconds)
            
            self.send_response(200)
            self.send_header('Content-Type', 'text/plain')
            self.send_header('Cache-Control', 'no-cache')
            self.end_headers()
            self.wfile.write(f'Delayed response ({delay_ms}ms)'.encode())
            return
        
        # 1MB file for throughput testing
        if path == '/fixed/1mb.bin':
            self.send_response(200)
            self.send_header('Content-Type', 'application/octet-stream')
            self.send_header('Content-Length', '1048576')
            self.send_header('Cache-Control', 'no-cache, no-store, must-revalidate')
            self.send_header('Pragma', 'no-cache')
            self.send_header('Expires', '0')
            self.end_headers()
            
            # Send 1MB of zeros
            chunk_size = 8192
            total_sent = 0
            while total_sent < 1048576:
                remaining = 1048576 - total_sent
                chunk = min(chunk_size, remaining)
                self.wfile.write(b'\x00' * chunk)
                total_sent += chunk
            return
        
        # Default response
        if path == '/' or path == '/index.html':
            self.send_response(200)
            self.send_header('Content-Type', 'text/html')
            self.end_headers()
            html = '''<!DOCTYPE html>
<html>
<head>
    <title>Network QoE Test Target</title>
</head>
<body>
    <h1>Network QoE Test Target Server</h1>
    <p>This server provides test endpoints for network quality measurements.</p>
    <ul>
        <li><a href="/health">Health Check</a> - Fast response endpoint</li>
        <li><a href="/slow?ms=2000">Slow Endpoint</a> - Configurable delay endpoint</li>
        <li><a href="/fixed/1mb.bin">1MB Test File</a> - For throughput testing</li>
    </ul>
</body>
</html>'''
            self.wfile.write(html.encode())
            return
        
        # 404 for other paths
        self.send_response(404)
        self.send_header('Content-Type', 'text/plain')
        self.end_headers()
        self.wfile.write(b'Not Found')

if __name__ == '__main__':
    PORT = int(os.environ.get('PORT', 80))
    with socketserver.TCPServer(("", PORT), TestHandler) as httpd:
        print(f"Test target server running on port {PORT}")
        httpd.serve_forever()