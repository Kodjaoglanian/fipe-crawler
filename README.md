# FIPE Crawler API

API REST escrita em **Go** para extrair, persistir e consultar dados da [Tabela FIPE](https://veiculos.fipe.org.br/) (preços médios de veículos no Brasil).

> Refatoração completa do projeto original em PHP (Symfony + AngularJS + MySQL) para uma arquitetura backend pura em Go com PostgreSQL.

---

## Sumário

- [Stack](#stack)
- [Arquitetura](#arquitetura)
- [Como rodar (Docker)](#como-rodar-docker)
- [Como rodar (local)](#como-rodar-local)
- [Variáveis de ambiente](#variáveis-de-ambiente)
- [Schema do banco](#schema-do-banco)
- [Endpoints](#endpoints)
- [Fluxo recomendado](#fluxo-recomendado)
- [Exemplos completos](#exemplos-completos)
- [Tipos de veículo e combustível](#tipos-de-veículo-e-combustível)
- [Testes](#testes)
- [Estrutura de diretórios](#estrutura-de-diretórios)
- [Licença](#licença)

---

## Stack

| Camada | Tecnologia |
|--------|-----------|
| Linguagem | Go 1.26+ |
| HTTP framework | [Gin](https://github.com/gin-gonic/gin) |
| Banco de dados | PostgreSQL 16 |
| Driver DB | [pgx/v5](https://github.com/jackc/pgx) (`pgxpool`) |
| Configuração | [godotenv](https://github.com/joho/godotenv) |
| Testes | `testing` + [testify](https://github.com/stretchr/testify) |
| Containerização | Docker + Docker Compose |

---

## Arquitetura

```
┌──────────┐       ┌─────────────────────────┐       ┌──────────────┐
│  Cliente │──────▶│  API (Gin) :8080        │──────▶│  PostgreSQL  │
└──────────┘       │                         │       │   :5432      │
                   │  ┌───────────────────┐  │       └──────────────┘
                   │  │  handlers         │  │
                   │  ├───────────────────┤  │       ┌──────────────┐
                   │  │  crawler ─────────┼──┼──────▶│  FIPE API    │
                   │  ├───────────────────┤  │       │  (HTTPS)     │
                   │  │  repository       │  │       └──────────────┘
                   │  └───────────────────┘  │
                   └─────────────────────────┘
```

**Camadas:**

- **`cmd/api`** — entrypoint: carrega configuração, conecta no Postgres (com retry), roda migrations, inicia o servidor HTTP com graceful shutdown.
- **`internal/handlers`** — handlers HTTP (Gin). Recebem requisições, validam parâmetros e orquestram crawler + repository.
- **`internal/crawler`** — cliente HTTP que consome a API pública da FIPE (`veiculos.fipe.org.br/api/veiculos/...`). Faz POST `application/x-www-form-urlencoded` com headers idênticos ao site oficial.
- **`internal/repository`** — acesso ao banco via `pgxpool`. Faz batch insert com `ON CONFLICT (fipe_cod, anomod, comb_cod) DO NOTHING` para idempotência.
- **`internal/models`** — structs (`Vehicle`, `Brand`, `Model`, `Table`, etc.).
- **`internal/config`** — carrega env vars com fallback para defaults.
- **`migrations/`** — SQL aplicado na inicialização (schema da tabela `veiculo`).

---

## Como rodar (Docker)

**Requisito:** Docker + Docker Compose v2.

```bash
# Build e sobe API + PostgreSQL
sudo docker compose up --build -d

# Verifica que está rodando
curl http://localhost:8080/

# Acompanhar logs
sudo docker compose logs -f app

# Parar
sudo docker compose down

# Parar e apagar dados do banco
sudo docker compose down -v
```

| Serviço | Porta host | Descrição |
|---------|-----------|-----------|
| `app` | `8080` | API Go |
| `db` | `5432` | PostgreSQL (user: `fipe`, pass: `fipe`, db: `fipe`) |

A API espera o banco ficar pronto e tenta reconectar até 10× com 2s de intervalo.

---

## Como rodar (local)

**Requisitos:** Go 1.26+, PostgreSQL rodando.

```bash
# 1. Crie o banco
createdb -U postgres fipe

# 2. Configure variáveis
cp .env.example .env
# edite DATABASE_URL conforme seu setup

# 3. Baixe dependências e rode
go mod download
go run ./cmd/api
```

A migration roda automaticamente no start.

---

## Variáveis de ambiente

| Var | Default | Descrição |
|-----|---------|-----------|
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/fipe?sslmode=disable` | Connection string PostgreSQL |
| `PORT` | `8080` | Porta HTTP |

Arquivo `.env` é carregado automaticamente se existir (via `godotenv`).

---

## Schema do banco

```sql
CREATE TABLE veiculo (
    id          SERIAL PRIMARY KEY,
    fipe_cod    VARCHAR(10),
    tabela_id   INTEGER NOT NULL,
    anoref      SMALLINT NOT NULL,    -- ano de referência
    mesref      SMALLINT NOT NULL,    -- mês de referência (1-12)
    tipo        SMALLINT NOT NULL,    -- 1=carro, 2=moto, 3=caminhão
    marca_id    INTEGER NOT NULL,
    marca       VARCHAR(50),
    modelo_id   INTEGER NOT NULL,
    modelo      VARCHAR(255) NOT NULL,
    anomod      SMALLINT NOT NULL,    -- ano do modelo (32000 = 0km)
    comb_cod    SMALLINT NOT NULL,    -- código do combustível
    comb_sigla  CHAR(1) NOT NULL,     -- G, A, D, F
    comb        VARCHAR(10) NOT NULL,
    valor       INTEGER NOT NULL      -- preço em reais (sem casas decimais)
);

CREATE UNIQUE INDEX veiculo_fipe_cod_anomod_comb_cod
    ON veiculo (fipe_cod, anomod, comb_cod);
```

---

## Endpoints

### Health

#### `GET /`
Health check.
```json
{ "msg": "FIPE Crawler API" }
```

---

### Consultas live na FIPE (não persiste)

#### `GET /tabelas`
Lista todas as tabelas de referência disponíveis na FIPE (uma por mês).

**Resposta:**
```json
[
  { "id": 332, "lbl": "abril/2026 ", "ano": "2026", "mes": "04" },
  { "id": 331, "lbl": "março/2026 ", "ano": "2026", "mes": "03" }
]
```

#### `GET /marcas?tabela_id={id}&tipo={1|2|3}`
Lista marcas para uma tabela e tipo de veículo.

**Resposta:**
```json
[
  { "id": 1, "label": "Acura", "tipo": 1 },
  { "id": 2, "label": "Agrale", "tipo": 1 }
]
```

#### `GET /modelos?tabela_id={id}&tipo={tipo}&marca_id={id}`
Lista modelos de uma marca.

**Resposta:**
```json
[
  { "id": 1, "label": "Integra GS 1.8", "tipo": 1 },
  { "id": 2, "label": "Legend 3.2/3.5", "tipo": 1 }
]
```

---

### Extração e persistência

#### `POST /extrair/marcas`
Retorna marcas (atalho para o GET equivalente, mas via POST).

**Body:**
```json
{ "tabela_id": 332, "tipo": 1 }
```

#### `POST /extrair/modelos`
**Body:**
```json
{ "tabela_id": 332, "tipo": 1, "marca_id": 1 }
```

#### `POST /extrair/veiculos`
Extrai **todos os veículos de uma marca específica** (todos os modelos × todos os anos × todos os combustíveis) e salva no banco.

**Body:**
```json
{ "tabela_id": 332, "tipo": 1, "marca_id": 1 }
```

**Resposta:**
```json
{ "saved": 432 }
```

⚠️ **Demorado** — pode levar minutos por marca, dependendo da quantidade de modelos.

#### `POST /extrair/tudo`
Extrai **todos os veículos** de uma tabela inteira para um tipo (todas as marcas).

**Body:**
```json
{ "ano": "2026", "mes": "4", "tipo": 1 }
```

**Resposta:**
```json
{
  "tabela_id": 332,
  "periodo": "04/2026",
  "tipo": 1,
  "total": 27251
}
```

⚠️ **Muito demorado** — pode levar **horas** (a FIPE limita requests). Para `tipo=1` (carros) é comum levar 1h+.

A inserção usa `ON CONFLICT DO NOTHING` na chave `(fipe_cod, anomod, comb_cod)`, então rodar duas vezes não duplica registros.

---

### Consultas no banco

#### `GET /tabelas/salvas`
Tabelas (período + tipo) já extraídas e salvas no banco.

**Resposta:**
```json
{
  "results": [
    { "id": "332-1", "lbl": "abril/2026 - carro" }
  ]
}
```

#### `GET /veiculos?tabela_id={id}&tipo={tipo}`
Lista veículos salvos de uma tabela e tipo.

**Resposta:**
```json
[
  {
    "id": 1,
    "fipe_cod": "038003-2",
    "tabela_id": 332,
    "anoref": 2026,
    "mesref": 4,
    "tipo": 1,
    "marca_id": 1,
    "marca": "Acura",
    "modelo_id": 1,
    "modelo": "Integra GS 1.8",
    "anomod": 1992,
    "comb_cod": 1,
    "comb_sigla": "G",
    "comb": "Gasolina",
    "valor": 14852
  }
]
```

#### `GET /veiculos/csv?tabela_id={id}&tipo={tipo}`
Faz download do CSV com todos os veículos da tabela/tipo.

```bash
curl "http://localhost:8080/veiculos/csv?tabela_id=332&tipo=1" -o fipe.csv
```

#### `GET /veiculos/search?q={termo}`
Busca veículos no banco por marca, modelo ou código FIPE (`ILIKE`).

```bash
curl "http://localhost:8080/veiculos/search?q=civic"
```

---

## Fluxo recomendado

1. **Liste as tabelas disponíveis** na FIPE:
   ```bash
   curl http://localhost:8080/tabelas
   ```

2. **Extraia tudo** para o período/tipo desejado (demorado):
   ```bash
   curl -X POST http://localhost:8080/extrair/tudo \
     -H "Content-Type: application/json" \
     -d '{"ano":"2026","mes":"4","tipo":1}'
   ```

3. **Consulte ou exporte** os dados salvos:
   ```bash
   curl "http://localhost:8080/veiculos/csv?tabela_id=332&tipo=1" -o fipe_abril_2026_carros.csv
   ```

---

## Exemplos completos

### Listar tabelas, escolher uma e extrair só uma marca

```bash
# Tabela mais recente
TABELA=$(curl -s http://localhost:8080/tabelas | jq '.[0].id')

# Marcas de carro
curl -s "http://localhost:8080/marcas?tabela_id=$TABELA&tipo=1" | jq

# Extrai todos os Honda da tabela mais recente (marca_id=25 — verificar)
curl -X POST http://localhost:8080/extrair/veiculos \
  -H "Content-Type: application/json" \
  -d "{\"tabela_id\":$TABELA,\"tipo\":1,\"marca_id\":25}"
```

### Buscar todos os Civic salvos

```bash
curl "http://localhost:8080/veiculos/search?q=civic" | jq
```

---

## Tipos de veículo e combustível

### Tipos
| Código | Tipo |
|-------|------|
| 1 | Carro |
| 2 | Moto |
| 3 | Caminhão |

### Combustíveis
| Código | Sigla | Nome |
|--------|-------|------|
| 1 | G | Gasolina |
| 2 | A | Álcool |
| 3 | D | Diesel |
| 4 | F | Flex |

### Ano modelo especial
- `32000` = veículo **0 km** (zero quilômetros / ano corrente).

---

## Script interativo

`fipe.sh` é um menu em bash que cobre **todas** as operações da API:

```bash
./fipe.sh
```

Requisitos: `curl` e `jq`.

**Menu:**

```
Consultas live (FIPE):
  1) Health check
  2) Listar tabelas FIPE
  3) Listar marcas
  4) Listar modelos

Extração / persistência:
  5) Extrair veículos de uma marca
  6) Extrair tudo de um período (ano/mês/tipo)
  7) Baixar TUDO (histórico completo)

Banco de dados:
  8) Tabelas salvas
  9) Listar veículos salvos
 10) Buscar veículos
 11) Exportar CSV

Utilitários:
 12) Smoke test
 13) Status containers
 14) Logs Docker
```

A opção **7** (baixar tudo) permite limitar últimos N meses e escolher tipos. Gera log com timestamp em `fipe_download_YYYYMMDD_HHMMSS.log` e é idempotente (pode interromper e rodar de novo).

## Testes unitários

```bash
go test ./...
```

---

## Estrutura de diretórios

```
fipe-crawler/
├── cmd/
│   └── api/
│       └── main.go              # entrypoint do servidor
├── internal/
│   ├── config/
│   │   └── config.go            # carga de env vars
│   ├── crawler/
│   │   ├── crawler.go           # cliente HTTP da FIPE
│   │   └── crawler_test.go
│   ├── handlers/
│   │   ├── handlers.go          # rotas Gin
│   │   └── handlers_test.go
│   ├── models/
│   │   └── models.go            # structs do domínio
│   └── repository/
│       └── repository.go        # acesso ao Postgres
├── migrations/
│   └── 001_initial_schema.sql   # schema aplicado no boot
├── .env.example
├── docker-compose.yml
├── Dockerfile                   # multi-stage (builder + alpine)
├── go.mod / go.sum
├── fipe.sh                      # menu interativo (CLI)
└── README.md
```

---

## Licença

[MIT](LICENSE.md)
