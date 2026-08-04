# Architecture — How This Deployment Runs

## Overview

Temporal's server is split into internal services (frontend, history, matching,
worker), but the `temporalio/auto-setup` image bundles all of them into a
single container for simple deployments — that's what's running here. A
production cluster would typically run these as separate, independently
scaled services; this "all-in-one" mode is intended for development and small
workloads, which fits a Raspberry Pi + local network use case well.

## Containers

The stack is defined in `docker-compose.yml` and made of four containers on a
private Docker network called `temporal-network`:

```
                         LAN (192.168.1.0/24)
                                 |
              +------------------+------------------+
              |                                      |
        host:8080                               host:7233
              |                                      |
   +----------v----------+              +------------v------------+
   |     temporal-ui      |   gRPC     |        temporal          |
   |  (temporalio/ui)     +----------->+  (temporalio/auto-setup) |
   |  Web UI, port 8080    |            |  Server, port 7233       |
   +----------------------+            |  (frontend/history/      |
                                        |   matching/worker,       |
                                        |   all-in-one)            |
                                        +------------+-------------+
                                                      |
                                             +--------v---------+
                                             |  temporal-postgresql |
                                             |  (postgres:13)       |
                                             |  persistence store   |
                                             +-----------------------+
                                                      ^
                                                      |
                                        +-------------+-------------+
                                        |    temporal-admin-tools     |
                                        |  (temporalio/admin-tools)   |
                                        |  CLI container, no ports    |
                                        |  exposed — exec into it     |
                                        +-----------------------------+
```

### `temporal-postgresql` (image `postgres:13`)

The persistence layer. Stores all workflow history, task queues, namespaces,
and visibility (search) data in two logical databases (`temporal` and
`temporal_visibility`) inside the same Postgres instance. Data lives in the
named Docker volume `temporal_postgresql-data`, so it survives container
restarts and `docker compose down` (but not `docker compose down -v`).

Not exposed to the host or LAN — only reachable from other containers on
`temporal-network`.

### `temporal` (image `temporalio/auto-setup:latest`)

The Temporal server itself. On first boot, "auto-setup" automatically:
1. Waits for Postgres to be reachable
2. Creates the `temporal` and `temporal_visibility` databases and runs schema
   migrations
3. Registers the `default` namespace
4. Registers custom search attributes used by the UI/CLI

After that one-time setup it behaves like a normal Temporal server. It listens
on gRPC port **7233**, which is both how the UI talks to it and how any SDK
(Go/Java/Python/TypeScript/.NET/etc.) or the CLI connects to run/query
workflows. Port 7233 is published to the host and bound to all interfaces, so
it's reachable from the LAN too.

Startup behavior is influenced by a dynamic config file mounted from
`./dynamicconfig/development-sql.yaml` (see
[CONFIGURATION.md](./CONFIGURATION.md)).

### `temporal-ui` (image `temporalio/ui:latest`)

The web UI, a separate service (not bundled into the server binary in modern
Temporal versions). It talks to the server over gRPC (`TEMPORAL_ADDRESS=temporal:7233`
on the internal Docker network) and serves a browser UI on port **8080**,
explicitly published as `0.0.0.0:8080` so any device on the LAN can reach it,
not just `localhost` on this host.

### `temporal-admin-tools` (image `temporalio/admin-tools:latest`)

A CLI-only container with the `temporal` CLI pre-installed, pre-configured to
point at `temporal:7233`. It publishes no ports — you interact with it via
`docker exec`. This is the normal way to run ad-hoc `temporal` CLI commands
without installing the CLI on the host.

## Data flow (typical workflow execution)

1. A worker process (your application code, using a Temporal SDK) connects to
   `192.168.1.135:7233` and polls task queues.
2. A client (also SDK, or the CLI) starts a workflow by calling the server on
   7233.
3. The server persists the workflow's event history to Postgres and dispatches
   tasks to polling workers.
4. Workers execute activities/workflow code and report results back over
   7233; the server persists each state transition.
5. The UI (port 8080) queries the server over gRPC to display namespaces,
   workflows, and their event histories — it does not talk to Postgres
   directly.

## Networking summary

| Container | Internal network | Host-published port | Bind address |
|---|---|---|---|
| `temporal-postgresql` | `temporal-network` | none | — |
| `temporal` | `temporal-network` | `7233` | `0.0.0.0` (default) |
| `temporal-ui` | `temporal-network` | `8080` | `0.0.0.0` (explicit) |
| `temporal-admin-tools` | `temporal-network` | none | — |

Both published ports bind to all interfaces, and this host has no firewall
active (checked: `ufw` not installed, `iptables` INPUT policy is `ACCEPT`), so
both are reachable from any device on the `192.168.1.0/24` LAN.

## Persistence & restart behavior

- Container restarts (`docker compose restart`, host reboot with
  `restart: unless-stopped`) keep all workflow/namespace data — it lives in
  Postgres, not in the `temporal` container.
- `docker compose down` removes containers but keeps the `postgresql-data`
  volume — a subsequent `docker compose up -d` picks up right where it left
  off (auto-setup detects the schema already exists and skips it).
- `docker compose down -v` (or manually removing the `temporal_postgresql-data`
  volume) is destructive — it deletes all workflow history permanently.
