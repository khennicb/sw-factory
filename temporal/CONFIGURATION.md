# Configuration Reference

Field-by-field explanation of `docker-compose.yml` and the dynamic config
file, for anyone tuning this deployment later.

## `docker-compose.yml`

### `postgresql` service

```yaml
image: postgres:13
environment:
  POSTGRES_PASSWORD: temporal
  POSTGRES_USER: temporal
volumes:
  - postgresql-data:/var/lib/postgresql/data
```

- Credentials (`temporal`/`temporal`) are dev-grade defaults — fine on a
  trusted LAN with no port published to the host, not something to reuse
  elsewhere.
- No `POSTGRES_DB` is set, because `auto-setup` (the `temporal` service)
  creates the `temporal` and `temporal_visibility` databases itself on first
  boot.
- No port is published — Postgres is only reachable from other containers on
  `temporal-network`, never from the host or LAN. That's intentional.

### `temporal` service

```yaml
image: temporalio/auto-setup:latest
environment:
  - DB=postgres12                # tells auto-setup which DB driver/schema to use
  - DB_PORT=5432
  - POSTGRES_USER=temporal
  - POSTGRES_PWD=temporal
  - POSTGRES_SEEDS=postgresql    # Docker service name, resolved via internal DNS
  - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
ports:
  - "7233:7233"                  # gRPC frontend, published on all interfaces
volumes:
  - ./dynamicconfig:/etc/temporal/config/dynamicconfig
```

- `DB=postgres12` selects the schema/driver Temporal uses for any Postgres
  version >= 12 (the label is historical, it's not tied to exactly v12; we're
  running Postgres 13 here and it's the correct setting).
- `DYNAMIC_CONFIG_FILE_PATH` points at a file mounted from the host
  (`./dynamicconfig/development-sql.yaml`), letting you change server
  behavior without rebuilding the image — just edit the file and
  `docker compose restart temporal`.
- Port `7233:7233` with no explicit bind address means Docker publishes it on
  `0.0.0.0` by default — reachable from the LAN. To restrict it to
  localhost-only, change to `"127.0.0.1:7233:7233"`.

### `temporal-admin-tools` service

```yaml
image: temporalio/admin-tools:latest
environment:
  - TEMPORAL_ADDRESS=temporal:7233
  - TEMPORAL_CLI_ADDRESS=temporal:7233
stdin_open: true
tty: true
```

- No ports published — access only via `docker exec`.
- `stdin_open`/`tty` keep the container alive for interactive `exec` sessions
  even though its default command doesn't do meaningful work on its own.

### `temporal-ui` service

```yaml
image: temporalio/ui:latest
environment:
  - TEMPORAL_ADDRESS=temporal:7233
  - TEMPORAL_CORS_ORIGINS=http://localhost:3000
ports:
  - "0.0.0.0:8080:8080"
```

- `TEMPORAL_CORS_ORIGINS` is left at its default value from the upstream
  example compose file. It only matters if you build a separate web app on
  `localhost:3000` that calls the UI's API directly; it doesn't affect normal
  browser access to the UI itself. Add more origins (comma-separated) if
  needed later.
- `0.0.0.0:8080:8080` is explicit (rather than relying on Docker's default) to
  make the "reachable from the LAN" intent obvious in the file itself.

### Network & volumes

```yaml
networks:
  temporal-network:
    name: temporal-network
volumes:
  postgresql-data:
```

- A single bridge network isolates the stack's internal traffic; only the two
  explicitly published ports (7233, 8080) cross into the host/LAN.
- The named volume `postgresql-data` (full name `temporal_postgresql-data`,
  prefixed by the Compose project name `temporal`) is where all workflow data
  actually lives — see [OPERATIONS.md](./OPERATIONS.md#backup--restore-the-database).

## Dynamic config — `dynamicconfig/development-sql.yaml`

Mounted into the `temporal` container and read at startup (and can be
re-read without a full restart for some keys, though a restart is the
reliable way to apply changes here). Currently set:

```yaml
frontend.enableUpdateWorkflowExecution: true
frontend.enableUpdateWorkflowExecutionAsyncAccepted: true
system.enableActivityEagerExecution: true
system.enableEagerWorkflowStart: true
system.forceSearchAttributesCacheRefreshOnRead: true
```

These are the standard development-mode flags from Temporal's own example
deployment:

- **`frontend.enableUpdateWorkflowExecution`** / **`...AsyncAccepted`** — turn
  on the Workflow Update feature (synchronous request/response into a running
  workflow), which is off by default on some server versions.
- **`system.enableActivityEagerExecution`** / **`enableEagerWorkflowStart`** —
  latency optimizations that let a worker execute a task immediately upon
  starting it, skipping a round trip through the task queue matching engine.
  Good for local/dev responsiveness.
- **`system.forceSearchAttributesCacheRefreshOnRead`** — always refresh the
  search-attribute cache on read instead of relying on cache TTL, so
  attribute changes show up immediately in the UI. Costs a bit of extra
  latency, acceptable at this scale.

To change server behavior, edit this file and run:
```bash
sudo docker compose restart temporal
```

Full list of available dynamic config keys:
https://docs.temporal.io/references/dynamic-configuration
