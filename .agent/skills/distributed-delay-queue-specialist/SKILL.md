---
name: distributed-delay-queue-specialist
description: Deep knowledge of the Distributed Delay Queue (DDQ) architecture. Use when modifying core scheduling, worker pooling, or job lifecycle logic.
---

# DDQ Specialist Skill

## Core Architecture
- **Delay Storage**: Jobs are stored in a Redis Sorted Set (ZSET) where the score is the target timestamp.
- **Fetch Mechanism**: Workers use a Lua script to atomically fetch the next due job and move it to a "working" state (or set a visibility timeout).
- **Retry Logic**: Failed jobs are moved back to the ZSET with an exponential backoff score.
- **Watchdog**: A separate process monitors "stuck" jobs that have exceeded their visibility timeout without being acknowledged.

## Job Lifecycle
1. **PENDING**: Stored in ZSET `queue:{name}:delay`.
2. **ACTIVE**: Being processed by a worker. Marked in `queue:{name}:processing`.
3. **COMPLETED**: Successfully processed and removed.
4. **FAILED**: Retry limit exceeded, moved to Dead Letter Queue (DLQ).

## Operating Procedures
- **Scaling**: Increase worker count horizontally. Redis is the bottleneck; monitor CPU and memory.
- **Observability**: Track `queue_depth`, `processing_rate`, and `error_rate` metrics.
- **Schema Management**: Use Protobuf for job payloads to ensure cross-language compatibility.
