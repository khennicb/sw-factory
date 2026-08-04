# Operations Guide

All commands assume:
```bash
cd /home/bkhennic/factory/temporal
```
Docker was installed system-wide and the current user isn't in the `docker`
group, so every command below is run with `sudo`. (If you later add your user
to the `docker` group with `sudo usermod -aG docker $USER` and re-login,
`sudo` becomes unnecessary.)

## Start / stop / restart

```bash
sudo docker compose up -d          # start all containers (or resume after a stop)
sudo docker compose down           # stop and remove containers; data volume is kept
sudo docker compose restart        # restart all containers in place
sudo docker compose restart temporal-ui   # restart just one service
sudo docker compose stop           # stop containers without removing them
sudo docker compose start          # start previously-stopped containers
```

Containers are configured with `restart: unless-stopped`, so they also come
back automatically after a host reboot (as long as Docker's own service is
enabled, which the installer set up via systemd).

## Status & health

```bash
sudo docker compose ps                                   # container status/ports
sudo docker exec temporal-admin-tools temporal operator cluster health
curl -o /dev/null -s -w "%{http_code}\n" http://localhost:8080   # UI reachability
```

## Logs

```bash
sudo docker compose logs -f                # all services, follow
sudo docker compose logs -f temporal       # just the server
sudo docker compose logs -f temporal-ui    # just the UI
sudo docker compose logs --tail 100 temporal-postgresql
```

## Upgrading images

Images are pinned to `:latest` in `docker-compose.yml`, so "upgrade" means
re-pulling and recreating:

```bash
sudo docker compose pull       # fetch newer images for all services
sudo docker compose up -d      # recreate containers with the new images
```

Postgres schema migrations (if the new Temporal server version needs them) are
handled automatically by the `auto-setup` entrypoint on next boot. Still, treat
this as a real upgrade: check the Temporal server release notes before
jumping versions, and consider backing up the volume first (below).

## Backup / restore the database

Data lives in the named volume `temporal_postgresql-data`. To back it up:

```bash
# Simple: dump via pg_dump while the stack is running
sudo docker exec temporal-postgresql pg_dumpall -U temporal > temporal-backup-$(date +%F).sql

# Or: snapshot the whole volume (stack should be stopped for consistency)
sudo docker compose stop
sudo docker run --rm -v temporal_postgresql-data:/data -v "$PWD":/backup \
  alpine tar czf /backup/postgresql-data-$(date +%F).tar.gz -C /data .
sudo docker compose start
```

To restore the SQL dump into a fresh instance:
```bash
cat temporal-backup-YYYY-MM-DD.sql | sudo docker exec -i temporal-postgresql psql -U temporal
```

## Resetting everything (destructive)

Deletes all workflow history and namespaces permanently:
```bash
sudo docker compose down -v
sudo docker compose up -d     # auto-setup runs fresh, recreating the default namespace
```

## Common troubleshooting

| Symptom | Check |
|---|---|
| UI shows connection error | `sudo docker compose logs temporal` — server may still be starting or Postgres isn't ready yet |
| `temporal` container keeps restarting | `sudo docker compose logs temporal` — usually a Postgres connectivity or schema issue |
| UI unreachable from another LAN device but works on `localhost` | Confirm the host's IP hasn't changed (`hostname -I`); confirm port map is `0.0.0.0:8080->8080` via `docker compose ps` |
| Can't exec into admin-tools | Confirm container name with `sudo docker ps` — it's `temporal-admin-tools` |
| Need the CLI binary name | Modern images use `temporal`, not the older `tctl` |

## Useful one-liners

```bash
# Which images/versions are actually running
sudo docker inspect --format='{{.Config.Image}}' temporal temporal-ui temporal-admin-tools temporal-postgresql
sudo docker exec temporal temporal --version

# Disk used by the Postgres volume
sudo docker system df -v | grep -A2 postgresql-data

# Resolved effective compose config (env vars substituted)
sudo docker compose config
```
