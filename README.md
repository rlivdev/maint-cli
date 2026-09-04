# maint-cli

CLI for backup and restore of PostgreSQL and MinIO. Runs via a Docker image, no local dependencies beyond Docker.

## Commands

```sh
maint postgres backup   # backup postgres database
maint postgres restore  # restore postgres database
maint minio backup      # backup minio bucket
maint minio restore     # restore minio bucket
```

### postgres backup

```sh
maint postgres backup --user <user> --password <pass> --database <db>
```

Options:
- `--host` (default `127.0.0.1`)
- `--port` (default `5432`)
- `--user` (required)
- `--password` (required)
- `--database` (required)

Produces a dump at `./postgres/<db>_<timestamp>.dump`.

### postgres restore

```sh
maint postgres restore --user <user> --password <pass> --database <db> [--file <dump>]
```

`--file` is optional. When omitted, the most recent dump for the database is used.

### minio backup

```sh
maint minio backup --access-key <key> --secret-key <secret> --bucket <bucket>
```

Options:
- `--endpoint` (default `http://127.0.0.1:9000`)
- `--access-key` (required)
- `--secret-key` (required)
- `--bucket` (required)

Compresses into `./minio/<bucket>_<timestamp>.tar.gz`.

### minio restore

```sh
maint minio restore --access-key <key> --secret-key <secret> --bucket <bucket> [--file <tar.gz>]
```

Creates the bucket if it does not exist. `--file` is optional; when omitted, the most recent backup is used.

## Installation

Requires Docker installed and running.

Install from GitHub (one-liner). Requires root/sudo since the binary is installed to `/usr/local/bin`:

```sh
curl -fsSL https://raw.githubusercontent.com/rlivdev/maint-cli/main/install.sh | sudo sh
```

The `install.sh` script:
1. Verifies Docker is installed and the script runs as root.
2. Pulls the `rlivdev/maint-cli:latest` image from the registry (updates existing images).
3. Downloads the `maint` script to `/usr/local/bin`.
4. Fallback: if the pull fails and no local image exists, builds from local sources.

### Manual

```sh
git clone git@github.com:rlivdev/maint-cli.git
cd maint-cli
sudo ./install.sh
```

### Update

Run `install.sh` again to pull the latest `:latest` image and refresh the binary:

```sh
curl -fsSL https://raw.githubusercontent.com/rlivdev/maint-cli/main/install.sh | sudo sh
```

## Use without installing

```sh
make build
./maint --help
```

## Makefile

- `build` — builds the `rlivdev/maint-cli:latest` image