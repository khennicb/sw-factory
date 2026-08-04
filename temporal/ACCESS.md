# Access Guide

Two endpoints matter: the **Web UI** (human browsing) and the **gRPC frontend**
(port 7233, used by SDKs and the CLI). Both are reachable from this host and
from any device on the local network.

## Web UI

| From | URL |
|---|---|
| This host (Raspberry Pi) | http://localhost:8080 |
| Any device on the LAN | **http://192.168.1.135:8080** |

The UI lets you browse namespaces, running/completed workflows, event
histories, task queues, and workers, and start/terminate/signal workflows by
hand. No login is configured (this deployment has no auth — see
[Security notes](#security-notes)).

`192.168.1.135` is this host's current DHCP-assigned LAN IP (`rasp-pi-power`).
If your router reassigns it, re-check with `hostname -I` on the host and use
the new address. If your network resolves mDNS (`.local`) hostnames, you may
also be able to reach it via `http://rasp-pi-power.local:8080`, but this
wasn't verified and isn't guaranteed on every network/OS.

## gRPC frontend (SDKs, CLI)

Address: `192.168.1.135:7233` from other LAN devices, or `localhost:7233` /
`temporal:7233` (internal Docker DNS name) from this host.

### Connecting from an SDK

Example (TypeScript):
```ts
import { Connection, Client } from '@temporalio/client';

const connection = await Connection.connect({ address: '192.168.1.135:7233' });
const client = new Client({ connection });
```

Example (Python):
```python
from temporalio.client import Client

client = await Client.connect("192.168.1.135:7233")
```

Example (Go):
```go
c, err := client.Dial(client.Options{HostPort: "192.168.1.135:7233"})
```

Any machine on the `192.168.1.0/24` network can run workers or start workflows
against this address — this is what makes it usable as a shared local
Temporal instance rather than confined to just this host.

### Connecting from the CLI

The `temporal-admin-tools` container has the `temporal` CLI preinstalled and
pre-pointed at the server. Run commands via `docker exec`:

```bash
# List running workflows
sudo docker exec temporal-admin-tools temporal workflow list

# Describe the default namespace
sudo docker exec temporal-admin-tools temporal operator namespace describe --namespace default

# Cluster health check
sudo docker exec temporal-admin-tools temporal operator cluster health

# Interactive shell inside the tools container
sudo docker exec -it temporal-admin-tools sh
```

Alternatively, install the `temporal` CLI directly on any LAN machine
(https://docs.temporal.io/cli#install) and point it at the server:
```bash
temporal workflow list --address 192.168.1.135:7233
```

## Namespaces

A `default` namespace was created automatically during auto-setup — it's
ready to use immediately, no need to create one before running your first
workflow.

## Security notes

This deployment has **no authentication, no TLS, and no firewall
restriction** — anyone on the local network can view all workflow data, start
workflows, and administer namespaces via the UI or gRPC port. This is
appropriate for a trusted home/local network used for development, but:

- **Do not** port-forward 7233 or 8080 to the public internet as configured.
- **Do not** put sensitive production data through this instance.
- If you need multi-user access control, look into Temporal Cloud or adding
  mTLS + an auth proxy in front of this deployment — out of scope for this
  setup.
