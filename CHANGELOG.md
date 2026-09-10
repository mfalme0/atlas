# Changelog

All notable changes to Atlas will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-09-10

### Added

- Project structure and Go module initialization
- Basic API server with chi router
- Health, readiness, and info endpoints
- Node, topology, service, job, cluster, raft, chaos, benchmark, and incident API stubs
- Request ID middleware
- Request logging middleware
- Panic recovery middleware
- CORS middleware
- Configuration system with environment variable overrides
- Event bus architecture
- Data models for nodes, edges, services, incidents, jobs, workers, benchmarks
- CLI with status, nodes, topology, services, jobs, cluster, raft, benchmarks, and incidents commands
- Agent entry point
- Docker Compose with PostgreSQL and Redis
- Multi-stage Dockerfile for server
- Next.js frontend skeleton
- Makefile with dev, test, lint, benchmark, build, and docker targets
- GitHub Actions CI workflow
- LICENSE, CONTRIBUTING.md, SECURITY.md, CODE_OF_CONDUCT.md
