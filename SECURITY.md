# Security

## Reporting vulnerabilities

Email: security@wirescope.io (update this with your actual contact)

Please include:
- Description of the issue
- Steps to reproduce
- Affected versions
- Potential impact

Do not open public issues for security vulnerabilities.

## Security Features

### Authentication
- ✅ Bcrypt password hashing
- ✅ Configurable user credentials via environment variables
- ✅ Role-based access control (admin, operator, viewer)
- ✅ HttpOnly, Secure cookies
- ✅ CSRF protection

### TLS/HTTPS Support
- ✅ Built-in TLS 1.2+ support
- ✅ Configurable cipher suites
- ✅ Certificate-based authentication
- ✅ Automatic secure cookie flag when TLS enabled

### Rate Limiting
- ✅ Redis-backed distributed rate limiting
- ✅ Per-client rate limiting
- ✅ Configurable thresholds

## Configuration

### Setting Up Authentication

See [docs/AUTHENTICATION.md](docs/AUTHENTICATION.md) for detailed instructions.

**Quick start:**
```bash
# Set custom admin password
export ADMIN_PASSWORD="YourSecurePassword123"

# Or configure multiple users
export AUTH_USERS="admin:SecurePass123:admin,viewer:ViewPass456:viewer"
```

### Enabling HTTPS

```yaml
# config.yaml
server:
  tls:
    enabled: true
    cert_file: /path/to/cert.pem
    key_file: /path/to/key.pem
    min_version: "1.2"
```

### Enabling Redis Rate Limiting

```bash
export REDIS_URL="redis://localhost:6379"
export RATE_LIMIT_RPS=100
```

## Known issues

- ⚠️ **IMPORTANT:** Change default credentials before production deployment
- ⚠️ Self-signed certificates: For development only, use proper CA-signed certs in production
- ⚠️ In-memory rate limiting: Does not work across multiple instances (use Redis for production)

## Best practices

### Production Deployment

1. **Authentication**
   - Use strong, unique passwords (min 12 characters, mixed case, numbers, symbols)
   - Avoid using default usernames like "admin"
   - Enable multi-factor authentication (coming soon)
   - Rotate passwords regularly

2. **TLS/HTTPS**
   - Always enable TLS in production
   - Use certificates from trusted CA (Let's Encrypt, etc.)
   - Configure TLS 1.2+ minimum version
   - Disable weak cipher suites

3. **Rate Limiting**
   - Enable Redis-backed rate limiting for multi-instance deployments
   - Set appropriate limits based on expected traffic
   - Monitor rate limit metrics

4. **Network Security**
   - Run with least privilege (don't run as root)
   - Use firewall rules to restrict access
   - Consider VPN for probe connections
   - Isolate services using network segmentation

5. **Database Security**
   - Enable SSL for database connections
   - Use strong database passwords
   - Restrict database access to application only
   - Regular backups with encryption

6. **Monitoring**
   - Enable audit logging
   - Monitor failed authentication attempts
   - Set up alerts for suspicious activity
   - Regular security audits

7. **Updates**
   - Keep dependencies updated
   - Subscribe to security advisories
   - Test updates in staging first
   - Have rollback plan

### API Token Security

- Generate tokens with: `openssl rand -hex 32`
- Never commit tokens to version control
- Use environment variables or secrets management
- Rotate tokens periodically
- Revoke compromised tokens immediately

### Metrics & Monitoring

Monitor these security metrics:
- `ingest_auth_failures_total{reason="invalid_token"}` - Failed auth attempts
- `ingest_rate_limit_exceeded_total` - Rate limit violations
- `admin_login_attempts_total{status="failed"}` - Failed login attempts
