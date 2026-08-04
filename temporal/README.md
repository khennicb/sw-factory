# Temporal — Local Deployment

Temporal is a durable execution platform: you write ordinary code (workflows and
activities), and the Temporal server transparently persists progress, retries
failures, and recovers state after crashes. This directory contains a
self-hosted Temporal stack running on this machine via Docker Compose.

This is a **development/local deployment** — good for building and testing
workflows on the local network, not hardened for production or exposure to
the internet (see [ACCESS.md](./ACCESS.md#security-notes)).

## Documents in this directory

| File | Contents |
|---|---|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | What each container does, how they connect, data flow |
| [ACCESS.md](./ACCESS.md) | How to reach the UI, gRPC endpoint, and CLI, from this host and the LAN |
| [OPERATIONS.md](./OPERATIONS.md) | Day-to-day commands: start/stop, logs, backup, upgrade, troubleshooting |
| [CONFIGURATION.md](./CONFIGURATION.md) | Field-by-field explanation of `docker-compose.yml` and dynamic config |

## Quick facts

- **Host**: `rasp-pi-power` (Debian 13 "trixie", aarch64 / Raspberry Pi), LAN IP `192.168.1.135`
- **Deployed**: 2026-08-05
- **Server version**: Temporal Server 1.29.1 (`temporalio/auto-setup:latest`)
- **UI version**: Temporal UI 2.42.1 (`temporalio/ui:latest`)
- **Database**: PostgreSQL 13.23, data persisted in Docker volume `temporal_postgresql-data`
- **Compose project**: `temporal` (dir: `/home/bkhennic/sw-factory/temporal`)

## Fastest path to using it

- **Web UI** (browse namespaces, workflows, workers): http://192.168.1.135:8080
- **gRPC endpoint** for SDKs/CLI (from another machine on the LAN): `192.168.1.135:7233`
- **CLI**, run from the host:
  ```bash
  sudo docker exec -it temporal-admin-tools temporal workflow list
  ```

Full detail on all of the above is in [ACCESS.md](./ACCESS.md).

## Starting/stopping

```bash
cd /home/bkhennic/sw-factory/temporal
sudo docker compose up -d      # start (or resume) the stack
sudo docker compose down       # stop and remove containers (data volume persists)
sudo docker compose ps         # check status
```

See [OPERATIONS.md](./OPERATIONS.md) for the complete command reference.
