# Installed dependencies log

Running log of everything Claude Code has installed on this machine for the
sw-factory project. Newest entry last.

<!-- Example entry:
## 2026-08-05
- **What:** ripgrep 14.1.0
- **Why:** needed for fast codebase search in the build script
- **Command:** `sudo apt-get install -y ripgrep`
-->

## 2026-08-05
- **What:** Go 1.26.5 (linux/arm64), official tarball from go.dev
- **Why:** sw-factory Step 1 (instructions/implem_1.txt) requires the Go
  Temporal SDK for services/workflow-engine and every activity shim; no Go
  toolchain existed on this host. Tarball chosen over `apt install golang-go`
  to get the current upstream release rather than Debian's lagging package.
- **Command:**
  ```bash
  curl -s -L -o go.tar.gz https://go.dev/dl/go1.26.5.linux-arm64.tar.gz
  tar -C /home/bkhennic -xzf go.tar.gz
  mv /home/bkhennic/go /home/bkhennic/go-sdk
  ```
- **Where:** extracted to `~/go-sdk` (`GOROOT`), `GOPATH=~/go`, both added to
  `PATH` via `~/.bashrc` and `~/.profile`. See
  `docs/step-1-workflow-engine.md` for full context.
