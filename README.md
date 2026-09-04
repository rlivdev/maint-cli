# maint-cli

CLI para backup e restore de PostgreSQL e MinIO. Roda via imagem Docker, sem dependencias locais alem do Docker.

## Comandos

```sh
maint postgres backup   # backup banco postgres
maint postgres restore  # restore banco postgres
maint minio backup      # backup bucket minio
maint minio restore     # restore bucket minio
```

### postgres backup

```sh
maint postgres backup --user <user> --password <pass> --database <db>
```

Opcoes:
- `--host` (default `127.0.0.1`)
- `--port` (default `5432`)
- `--user` (obrigatorio)
- `--password` (obrigatorio)
- `--database` (obrigatorio)

Gera dump em `./postgres/<db>_<timestamp>.dump`

### postgres restore

```sh
maint postgres restore --user <user> --password <pass> --database <db> [--file <dump>]
```

`--file` opcional. Omitido, usa dump mais recente do banco.

### minio backup

```sh
maint minio backup --access_key <key> --secret_key <secret> --bucket <bucket>
```

Opcoes:
- `--endpoint` (default `http://127.0.0.1:9000`)
- `--access_key` (obrigatorio)
- `--secret_key` (obrigatorio)
- `--bucket` (obrigatorio)

Compacta em `./minio/<bucket>_<timestamp>.tar.gz`

### minio restore

```sh
maint minio restore --access_key <key> --secret_key <secret> --bucket <bucket> [--file <tar.gz>]
```

Cria bucket se nao existe. `--file` opcional, usa backup mais recente.

## Instalacao

Requer Docker instalado e rodando.

Instalar via GitHub (one-liner):

```sh
curl -fsSL https://raw.githubusercontent.com/rlivdev/maint-cli/main/install.sh | sh
```

O `install.sh`:
1. Verifica se Docker esta instalado.
2. Baixa a imagem `rlivdev/maint-cli:latest` do registry (atualiza imagens ja existentes).
3. Copia o script `maint` para `/usr/local/bin`.
4. Fallback: se o pull falhar e nao houver imagem local, constroi a partir do codigo local.

### Manual

```sh
git clone git@github.com:rlivdev/maint-cli.git
cd maint-cli
./install.sh
```

### Atualizar

Rode `install.sh` novamente para puxar a ultima imagem `:latest` e atualizar o binario:

```sh
curl -fsSL https://raw.githubusercontent.com/rlivdev/maint-cli/main/install.sh | sh
```

## Usar sem instalar

```sh
make build
./maint --help
```

## Makefile

- `build` — constroi imagem `rlivdev/maint-cli`
- `postgres-backup` / `postgres-restore` / `minio-backup` / `minio-restore` — exemplos com credenciais demo
