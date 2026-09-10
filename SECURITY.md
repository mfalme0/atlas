# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Atlas, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, email security reports to the maintainers directly.

Please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

## Response Timeline

- Acknowledgment within 48 hours
- Initial assessment within 1 week
- Fix or mitigation plan within 2 weeks for critical issues

## Scope

Atlas is an infrastructure intelligence engine. Security concerns include:

- API authentication and authorization
- Secret handling
- Container and Kubernetes integration security
- Input validation
- Command injection prevention
- Network communication security

## Best Practices

- Never commit secrets or credentials
- Use environment variables for configuration
- Validate all user inputs
- Use HTTPS in production
- Enable authentication for API endpoints
- Review chaos experiment permissions
