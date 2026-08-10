# Brekke CLI

Ferramenta de linha de comando para **backup** e **restore** de **PostgreSQL** e **MinIO**, encapsulada em um container Docker. Foco: simplicidade, alta usabilidade e configuração direta.

## Requisitos

- **Docker** instalado e funcionando
- **Go 1.26.5+** (apenas para build/desenvolvimento local)

## Instalação

### 1. Build da imagem

```sh
cd brekke-cli
make docker-build          # gera imagem brekke:latest
```

### 2. Instalar o wrapper

```sh
# compila o binário nativo (usado apenas como wrapper)
make build                 # gera ./bin/brekke

# opcional: colocar no PATH
install bin/brekke ~/.local/bin/brekke
```

O binário nativo apenas valida argumentos e dispara o container Docker. Toda a lógica roda no container.

## Configuração

### Estrutura de diretórios

```
~/.brekke/
├── profiles/            # perfis de configuração (YAML)
└── backups/             # pacotes de backup gerados
    └── <profile>/
        └── <profile>-<YYYYMMDD-HHMMSS>.tar.gz
```

O diretório `~/.brekke` é montado como volume no container (`/data`). Se não existir, é criado automaticamente pelo wrapper na primeira execução.

### Criando um perfil

O perfil é identificado pelo **nome do arquivo** (sem a extensão `.yaml`). Crie o arquivo:

```sh
cp examples/sample-service.yaml ~/.brekke/profiles/sample-service.yaml
```

Exemplo de perfil:

```yaml
version: "1"
name: "sample-service"

backup:
  postgres:
    host: "db.sample.internal"
    port: 5432
    user: "postgres"
    password: "MinhaSenhaSuperSegura123"
    database: "sample_prod"

  minio:
    host: "minio.sample.internal"
    port: "9000"
    access_key: "minioadmin"
    secret_key: "minioadminpassword"
    buckets:
      - "sample"
      - "sample2"
```

| Campo | Descrição |
|-------|-----------|
| `version` | Versão do formato do perfil (atual: `"1"`) |
| `name` | Nome do perfil. Deve ser igual ao nome do arquivo |
| `backup.postgres.*` | Parâmetros de conexão PostgreSQL |
| `backup.minio.*` | Parâmetros de conexão MinIO. `buckets` vazio ou ausente = todos os buckets |

**Portas padrão** (se não informadas): PostgreSQL `5432`, MinIO `9000`.

> Configure apenas `backup.postgres` **ou** `backup.minio` para fazer backup de um único recurso.

## Uso

### Backup

```sh
brekke backup                          # usa o profile padrão
brekke backup -p sample-service        # profile informado por flag
brekke backup --profile sample-service # forma longa
```

Resultado:

```
Backup concluído: /data/backups/sample-service/sample-service-20260810-120000.tar.gz
```

### Restore

```sh
brekke restore                          # restaura o backup mais recente do profile padrão
brekke restore -p sample-service        # restaura o backup mais recente do perfil
brekke restore -f sample-service-20260810-120000.tar.gz
brekke restore --file sample-service-20260810-120000.tar.gz
```

### Flags

| Flag | Shorthand | Comando | Descrição |
|------|-----------|---------|-----------|
| `--profile` | `-p` | backup/restore | Nome do perfil |
| `--file` | `-f` | restore | Nome do pacote a restaurar |
| `--data-dir` | `-d` | todos | Diretório raiz de dados (padrão: `/data` no container) |

> **Aviso:** o restore sobrescreve os dados atuais do PostgreSQL e MinIO do perfil. Use com cautela.

## Como funciona (resumo)

1. **Wrapper** no host cria `~/.brekke` se necessário e dispara o container com volume `~/.brekke:/data`, repassando flags e argumentos.
2. **Container** (Go + cobra) resolve o profile e valida a `version`.
3. **Backup**:
   - PostgreSQL → `pg_dump` do banco.
   - MinIO → `mc mirror` dos buckets.
   - Tudo empacotado em um único `.tar.gz`.
4. **Restore**: descompacta o `.tar.gz` e importa de volta.

## Contêiner (imagem Docker)

A imagem contém:

- Binário `brekke` (Go + cobra)
- `pg_dump` / `psql` (cliente PostgreSQL)
- `mc` (MinIO Client)
- `tar` / `gzip`

## Desenvolvimento

```sh
make test        # roda testes
make vet         # análise estática
make build       # build local (bin/brekke)
make docker-build
```

Estrutura do código:

```
main.go                 # entrypoint
cmd/                    # cobra commands (backup, restore)
internal/brekke/        # lógica de domínio
  profile.go            # carregamento e validação de perfis
  backup.go             # backup (pg_dump, mc, empacotamento)
```
