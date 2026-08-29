# Maint CLI — Especificação Técnica

## 1. Visão Geral

`maint-cli` = ferramenta de linha de comando encapsulada em container Docker. Executa operações de **backup** e **restore** de PostgreSQL e MinIO com foco em simplicidade, alta usabilidade e configuração direta.

- Linguagem: **Go** (versão recente)
- CLI: **cobra** (versão recente)
- Execução: wrapper no host dispara container com volume montado

## 2. Arquitetura

```
+----------------------------- Host -----------------------------+
|  ~/.maint/                 |                                  |
|   ├── profiles/*.yaml       |     maint (wrapper binário)      |
|   └── backups/<profile>/    └─────────┐                          |
+---------------------------------------┼-------------------------+
                                        | docker run -v ~/.maint:/data
                                  +------v-----------------------+
                                  |  Container maint            |
                                  |   - código Go + cobra        |
                                  |   - pg_dump / psql           |
                                  |   - minio CLI (mc)           |
                                  +-----------------------------+
```

### Componentes

| Camada | Responsabilidade |
|--------|-----------------|
| **Wrapper (host)** | Valida args, resolve profile, monta comando `docker run`, monta volume `~/.maint:/data`, passa flags e args como variáveis de ambiente / args do container |
| **CLI Go + cobra** | Executa comandos `backup` e `restore`, parse de flags/profiles, orquestração pg_dump + mc, geração/leitura do pacote `.tar.gz` |
| **Ferramentas** | `pg_dump`/`psql` (PostgreSQL), `mc` (MinIO), `tar` (empacotamento) |

## 3. Estrutura de Diretórios

```
~/.maint/
├── profiles/            # arquivos de configuração YAML (1 arquivo = 1 profile)
└── backups/             # pacotes de backup
    └── <profile-name>/
        └── <profile-name>-<YYYYMMDD-HHMMSS>.tar.gz
```

| Diretório | Uso |
|-----------|-----|
| `~/.maint/profiles` | Leitura de configuração (montado no container) |
| `~/.maint/backups/<profile>` | Gravação/generating e restauração de pacotes (montado no container) |

Volume único `~/.maint:/data` no container dá acesso a profiles e backups.

## 4. Perfis (Profiles)

- Configuração em arquivos **YAML por perfil**, um arquivo por profile.
- **Profile identificado pelo nome do arquivo** (sem extensão).
- Se arquivo inexistente → erro informando que o profile não foi encontrado.
- Credenciais e parâmetros de conexão **contidos no arquivo de perfil** — sem gerenciamento manual de variáveis de ambiente no container.

### 4.1 Formato do Arquivo

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

### 4.2 Campos

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|-------------|-----------|
| `version` | string | sim | Versão do formato do arquivo. Validação de compatibilidade |
| `name` | string | sim | Nome do profile (deve casar com nome do arquivo) |
| `backup.postgres.host` | string | sim* | Host PostgreSQL |
| `backup.postgres.port` | int | sim* | Porta PostgreSQL (padrão 5432) |
| `backup.postgres.user` | string | sim* | Usuário PostgreSQL |
| `backup.postgres.password` | string | sim* | Senha PostgreSQL |
| `backup.postgres.database` | string | sim* | Banco PostgreSQL |
| `backup.minio.host` | string | não** | Host MinIO |
| `backup.minio.port` | string/int | não** | Porta MinIO (padrão 9000) |
| `backup.minio.access_key` | string | não** | Access key MinIO |
| `backup.minio.secret_key` | string | não** | Secret key MinIO |
| `backup.minio.buckets` | array de string | não** | Buckets a incluir (vazio/ausente = todos) |

\* obrigatório se recurso PostgreSQL presente
\*\* obrigatório se recurso MinIO presente

### 4.3 Versionamento de Perfil

- Campo `version` controla compatibilidade.
- Versão atual: `"1"`.
- Ao ler arquivo, validar versão suportada. Versão futura/desconhecida → erro de compatibilidade claro.

## 5. Comandos

### 5.1 Backup

```
maint backup
maint backup -p sample-service
maint backup --profile sample-service
```

- Sem flag `-p/--profile`: usar profile padrão (definir convenção, ex. `default` ou exigir flag).
- Backup de **todos os recursos** por padrão (postgres + minio).
- Fluxo:
  1. Resolver profile (flag ou padrão).
  2. Carregar e validar YAML.
  3. Criar diretório de trabalho temporário no container.
  4. PostgreSQL → `pg_dump` do banco.
  5. MinIO → `mc mirror` dos buckets.
  6. Empacotar em `.tar.gz`.
  7. Salvar em `~/.maint/backups/<profile>/<profile>-<YYYYMMDD-HHMMSS>.tar.gz` (via volume `/data`).

### 5.2 Restore

```
maint restore
maint restore -p sample-service
maint restore --profile sample-service
maint restore -f sample-service-20260810-120000.tar.gz
maint restore --file sample-service-20260810-120000.tar.gz
```

- Sem `-f/--file`: usar pacote mais recente do profile.
- Fluxo:
  1. Resolver profile (flag ou padrão).
  2. Resolver arquivo (`-f` ou mais recente em `backups/<profile>`).
  3. Verificar existência do arquivo → erro se ausente.
  4. Descompactar `.tar.gz`.
  5. PostgreSQL → `psql` de importação.
  6. MinIO → `mc mirror` de upload.

### 5.3 Flags

| Flag | Shorthand | Comando | Descrição |
|------|-----------|---------|-----------|
| `--profile` | `-p` | backup/restore | Nome do profile |
| `--file` | `-f` | restore | Nome do pacote `.tar.gz` |

## 6. Formato do Pacote de Backup

- Arquivo: `<profile>-<YYYYMMDD-HHMMSS>.tar.gz`
- Exemplo: `sample-service-20260810-120000.tar.gz`
- Conteúdo (estrutura interna do tar):

```
<profile>/
├── postgres/
│   └── <database>.dump
└── minio/
    └── <bucket>/...   (ou buckets individuais)
```

- Carimbo de tempo `YYYYMMDD-HHMMSS` garante unicidade e ordenação temporal (usado para "mais recente").

## 7. Wrapper de Execução (Host)

- Binário leve no host (`maint`).
- Responsabilidades:
  1. Resolver profile e flags.
  2. Montar comando `docker run`:
     - Volume: `-v ~/.maint:/data`
     - Imagem do container com Go + cobra + `pg_dump` + `psql` + `mc` + `tar`.
  3. Passar comando/flags ao container.
  4. Propagate exit code e saída.
- Wrapper não contém lógica de negócio — apenas dispach avalia metadados de execução para o container.

Alternativa (a definir): wrapper em Go compilado ou script. Substituir ambiente dentro do container.

## 8. Tratamento de Erros

| Cenário | Comportamento |
|---------|---------------|
| Profile não encontrado | Erro: `profile "<nome>" não encontrado em ~/.maint/profiles` |
| Arquivo de profile com `version` não suportada | Erro de compatibilidade |
| YAML inválido | Erro de parse com detalhe |
| Arquivo de restore não encontrado | Erro informando arquivo ausente |
| Falha pg_dump / psql | Erro com saída da ferramenta, abortar operação |
| Falha mc | Erro com saída da ferramenta, abortar operação |

- Exit codes distintos: 0 sucesso, 1 erro genérico, 2 erro de validação de args/uso.

## 9. Segurança

- Senhas/credenciais **somente no arquivo de perfil**, não em variáveis de ambiente do container nem no processo host.
- Container recebe dados via volume montado — não embutir segredos na imagem.
- Recomendado: permissões restritas em `~/.maint/profiles` (ex. `chmod 600`).
- Não logar credenciais em saída padrão.

## 10. Requisitos do Container

Imagem base deve conter:
- Binário `maint` (Go + cobra)
- `pg_dump`, `psql` (cliente PostgreSQL)
- `mc` (MinIO Client)
- `tar` (+ `gzip`)

## 11. Roadmap / Pendências de Decisão

- [ ] Definir profile padrão quando `-p` ausente (ex. `default`).
- [ ] Estrutura interna exata do tar (postgres + minio dentro do profile).
- [ ] Wrapper: binário Go vs script de shell.
- [ ] Snapshot flags extras (ex. `--no-minio`, `--no-postgres`).
- [ ] Tratamento de buckets inexistentes no restore.