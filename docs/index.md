# Kashvi Documentation

Kashvi is a Laravel-inspired Go framework focused on practical defaults: routing, middleware, auth, ORM helpers, migrations, queue workers, scheduler, storage, and testing support in one codebase.

## Start Here

1. [Installation & Quick Start](./installation.md)
2. [CRUD Walkthrough](./crud.md)
3. [CLI Reference](./cli.md)

## Core Guides

| Guide | What you learn |
|---|---|
| [Configuration](./configuration.md) | `.env` and `config/app.json` loading and defaults |
| [Routing](./routing.md) | Named routes, groups, per-route middleware, route listing |
| [Context API](./context.md) | Request parsing, validation, JSON/file responses |
| [Validation](./validation.md) | Validation tags and error handling flow |
| [Authentication](./auth.md) | JWT helpers and role-based access flow |
| [ORM & Database](./orm.md) | Query builder API, pagination, parallel queries |
| [Migrations](./migrations.md) | Registering migrations and lifecycle commands |

## Runtime Systems

| Guide | What you learn |
|---|---|
| [Queue](./queue.md) | Background jobs, retries, delayed jobs, failed jobs |
| [Worker Pool](./workerpool.md) | Bounded concurrency for CPU/IO workloads |
| [Storage](./storage.md) | Local/S3-compatible disk abstraction |
| [WebSocket & SSE](./websocket.md) | Realtime events over WS and server-sent events |
| [gRPC](./grpc.md) | Running standalone gRPC server with health checks |
| [Logging](./logging.md) | Structured logs and optional MongoDB sink |
| [TestKit](./testkit.md) | Scenario-based API testing helpers |

## Suggested Learning Path

1. Complete [Installation & Quick Start](./installation.md).
2. Follow [CRUD Walkthrough](./crud.md) end-to-end.
3. Open [Routing](./routing.md), [Context API](./context.md), and [Validation](./validation.md) together when building handlers.
4. Add background processing with [Queue](./queue.md) and [Worker Pool](./workerpool.md).
5. Expand infrastructure with [Storage](./storage.md), [gRPC](./grpc.md), and [Logging](./logging.md).
