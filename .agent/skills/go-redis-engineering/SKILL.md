---
name: go-redis-engineering
description: Best practices for building Go services with Redis. Use when implementing new storage logic, workers, or API handlers.
---

# Go & Redis Engineering Skill

Expertise in building high-performance, reliable Go services backed by Redis.

## Go Standards
- **Interface Segregation**: Use interfaces for storage to allow easier mocking.
- **Dependency Injection**: Inject Redis clients and service dependencies.
- **Validation**: Use `validate` tags or manual checks for all incoming API requests.
- **Logging**: Use structured logging (e.g., `zap` or `slog`) with relevant fields (trace_id, job_id).

## Redis Best Practices
- **Lua Scripting**: Use Lua for any logic that requires atomicity across multiple keys or operations.
- **Pipeline**: Use pipelines for batch operations to reduce network RTT.
- **Visibility Timeout**: Implement a "fetch and hold" pattern (using Lua) to ensure jobs aren't lost if a worker crashes.
- **Key Design**: Use a consistent hierarchy: `ddq:{queue_name}:jobs:{job_id}`.

## Testing
- Use `miniredis` for fast unit tests.
- Use Docker for integration tests to ensure Lua scripts run correctly.
- Always check for Goroutine leaks in worker tests.
