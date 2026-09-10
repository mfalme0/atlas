# Contributing to Atlas

Thank you for your interest in contributing to Atlas.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Create a feature branch: `git checkout -b feature/my-feature`
4. Make your changes
5. Run tests: `make test`
6. Run linter: `make lint`
7. Commit your changes
8. Push to your fork and submit a Pull Request

## Development Setup

```bash
# Start dependencies
docker compose up -d postgres redis

# Run the server
make dev

# Run tests
make test
```

## Code Style

- Follow idiomatic Go conventions
- Run `gofmt -s -w .` before committing
- Run `go vet ./...` to check for issues
- All exported functions and types must have documentation comments
- Write tests for new functionality

## Pull Request Guidelines

- Keep PRs focused on a single change
- Include tests for new features
- Update documentation if needed
- Ensure CI passes
- Write clear commit messages

## Reporting Issues

Use GitHub Issues for bug reports and feature requests. Include:

- Steps to reproduce
- Expected behavior
- Actual behavior
- Go version and OS
