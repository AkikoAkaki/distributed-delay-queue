---
name: code-review
description: Reviews code changes for bugs, style issues, and best practices. Use when evaluating new features or bug fixes in the distributed-delay-queue project.
---

# Code Review Skill

When reviewing code in this project, follow these rigorous engineering standards:

## Review Checklist
1. **Correctness**: 
    - Does the logic align with the delay queue's distributed nature?
    - Are race conditions handled in Go and Redis?
2. **Redis Atomicity**:
    - Are multiple Redis operations encapsulated in Lua scripts where necessary?
    - Are keys correctly namespaced and prefixed?
3. **Concurrency & Context**:
    - Is `context.Context` used correctly for cancellation?
    - Are goroutines properly synced (channels, waitgroups)?
4. **Testing**:
    - Are there unit tests for the change?
    - Do tests use `gomock` for interfaces?
5. **Error Handling**:
    - Are errors wrapped with context?
    - Is the gRPC error code appropriate?

## How to Provide Feedback
- Provide specific line-by-line comments.
- Explain the "Why" behind suggestions (e.g., "Using ZADD here without Lua might lead to a race condition").
- Suggest specific code snippets for fixes.
