# Security

## Reporting vulnerabilities

Email: [your-email@example.com] (update this)

Please include:
- Description of the issue
- Steps to reproduce
- Affected versions
- Potential impact

Do not open public issues for security vulnerabilities.

## Known issues

- Default credentials (`admin/admin123`) - change these in production
- API token `demo-token` - for testing only
- No TLS by default - put nginx/caddy in front for HTTPS
- In-memory rate limiting - not shared across multiple ingest instances

## Best practices

- Use strong API tokens (not `demo-token`)
- Enable TLS for all endpoints
- Run with least privilege (don't run as root)
- Keep dependencies updated
- Review firewall rules
- Back up PostgreSQL regularly
