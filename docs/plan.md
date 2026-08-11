# Plano de Implementacao — Notification Service (Go)

**Status:** Aprovado
**Data:** 2026-08-11

## Decisoes de design

Cada decisao listada aqui tem um ADR completo em `docs/decisions/`. A tabela abaixo e um indice; os ADRs contem a justificativa completa e as alternativas rejeitadas.

| Decisao | Escolha | ADR |
|---|---|---|
| Arquitetura | Hexagonal leve, `internal/` sem `pkg/` exportavel | [0002](docs/decisions/adr-0002-lean-hexagonal-architecture.md) |
| Banco de dados | PostgreSQL via `pgx/v5`, SQL escrito a mao | [0003](docs/decisions/adr-0003-postgresql-pgx-hand-written-sql.md) |
| IDs | UUIDv7, prefixo so na borda HTTP (`ntf_`, `dly_`) | [0004](docs/decisions/adr-0004-uuidv7-identifiers.md) |
| Recipient | JSONB + `recipient_search` indexado | [0007](docs/decisions/adr-0007-polymorphic-recipient-jsonb.md) |
| Payload fingerprint | SHA-256 de campos canonicos normalizados | [0005](docs/decisions/adr-0005-idempotency-fingerprint.md) |
| Worker | `time.Ticker` + `SKIP LOCKED`, in-process | [0006](docs/decisions/adr-0006-in-process-polling-worker.md) |
| Autenticacao | Basic Auth, conta tecnica por ambiente | [0008](docs/decisions/adr-0008-basic-auth-technical-accounts.md) |
| Migrations | `golang-migrate` com `embed` | [0003](docs/decisions/adr-0003-postgresql-pgx-hand-written-sql.md) |
| Docker | Multi-stage `golang:1.24-alpine` -> `distroless/static` | — |
| Configuracao | `os.Getenv` puro, defaults documentados | — |
| Testes | Integracao = fluxo feliz com PostgreSQL real; Unitarios = todas variacoes com fakes | — |

## Estrutura do projeto

```
cmd/server/main.go
internal/
  domain/       # Notification, Delivery, Channel, Status, Recipient (tipos puros)
  app/          # ports (interfaces) + use cases
  http/         # handler, router, basic auth middleware, idcodec, dto
  postgres/     # repository adapter, migrations (embed)
  email/        # EmailProvider fake adapter
  worker/       # polling worker, lease claim loop
  config/       # env parsing
docs/
  spec.md
  plan.md
  decisions/    # ADRs
```

## Parametros do worker

| Parametro | Default | Descricao |
|---|---|---|
| `LEASE_DURATION` | 30s | Duracao da lease de envio |
| `POLL_INTERVAL` | 2s | Intervalo entre polls |
| `BATCH_SIZE` | 10 | Notificacoes reclamadas por batch |
| `MAX_CONCURRENCY` | 5 | Goroutines simultaneas de envio |

## Configuracao (env vars)

| Variavel | Required | Default | Descricao |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL DSN |
| `BASIC_AUTH_USERNAME` | no | `notification` | Usuario Basic Auth |
| `BASIC_AUTH_PASSWORD` | no | `notification` | Senha Basic Auth |
| `PORT` | no | `8080` | Porta HTTP |
| `LEASE_DURATION` | no | `30s` | Duracao da lease |
| `POLL_INTERVAL` | no | `2s` | Intervalo de poll |
| `BATCH_SIZE` | no | `10` | Tamanho do batch |
| `MAX_CONCURRENCY` | no | `5` | Concorrencia maxima |

Banco de dados: `notification_db` com usuario `notification_usr` (default local).

## Fases de implementacao

### Fase 1: Scaffold

**Objetivo:** Projeto compilavel, Docker funcional, health check.

- [x] `go.mod` (module `github.com/nexus-shopping/notification-service`)
- [x] Estrutura de diretorios
- [x] `internal/config/` — parse de env vars com validacao
- [x] `cmd/server/main.go` — esqueleto: config, logger, HTTP server com `/health`
- [x] `Dockerfile` — multi-stage com distroless
- [x] `docker-compose.yml` — PostgreSQL + servico
- [x] `.env.example`
- [x] `Makefile`

**Done when:** `docker compose up` sobe o servico e `curl /health` retorna `{"status":"ok"}` com o banco acessivel.

### Fase 2: Dominio

**Objetivo:** Todos os tipos puros do dominio, sem dependencias externas.

- [ ] `internal/domain/` — Notification, Delivery, Channel, Status, Recipient
- [ ] Transicoes de estado com fencing de lease (`PENDING -> SENDING -> SENT/FAILED`)
- [ ] Validacao de recipient por channel (email para EMAIL, phone_number para SMS)
- [ ] `internal/clock/` — interface Clock + implementacao real (injetavel)
- [ ] `internal/clock/id.go` — UUIDv7Generator (padrao dummy-pay)
- [ ] `internal/app/ports.go` — interfaces: NotificationRepository, DeliveryRepository, EmailProvider, IDGenerator, Clock

**Done when:** Testes unitarios de dominio passam (`-race -count=5`).

### Fase 3: Persistencia

**Objetivo:** PostgreSQL acessivel, migrations aplicadas, repositories funcionais.

- [ ] Migrations SQL (`embed`) — tabelas `notifications` e `notification_deliveries`
- [ ] `internal/postgres/` — NotificationRepository adapter
  - Insert com `ON CONFLICT (notification_key)` + fingerprint check
  - Claim atomico com `FOR UPDATE SKIP LOCKED` + lease token
  - Complete condicional ao lease token
  - Find by ID
- [ ] `internal/postgres/` — DeliveryRepository adapter
  - Insert delivery
  - Complete delivery condicional

**Done when:** Testes de integracao (fluxo feliz) passam com PostgreSQL real. Testes unitarios cobrem todas as variacoes de conflito e edge cases.

### Fase 4: HTTP API

**Objetivo:** API REST funcional com autenticacao e idempotencia.

- [ ] `internal/http/idcodec.go` — encode/decode com prefixos `ntf_` e `dly_`
- [ ] `internal/http/middleware.go` — Basic Auth middleware
- [ ] `internal/http/dto.go` — request/response DTOs com validacao
- [ ] `internal/http/handler.go` — `POST /v1/notifications`, `GET /v1/notifications/{id}`
- [ ] `internal/app/` — SendNotification use case, GetNotification use case
- [ ] Validacao: notification_key obrigatorio, channel EMAIL, email valido, subject/body obrigatorios
- [ ] Idempotencia: replay = 202 com existente; payload diferente = 409

**Done when:** Testes unitarios (`httptest`) cobrem auth, validacao, 202, 409, replay. Teste de integracao cobre POST + GET happy path.

### Fase 5: EmailProvider

**Objetivo:** Adapter fake de e-mail com log estruturado e falha controlada.

- [ ] `internal/email/` — EmailProvider fake
  - Sucesso padrao com log via `slog`
  - Falha deterministica por configuracao (ex: `X-Fail-Delivery: true` no delivery key)
  - Nunca loga corpo completo do e-mail
  - Preserva `deliveryKey` para rastreabilidade

**Done when:** Testes unitarios cobrem sucesso e falha.

### Fase 6: Worker

**Objetivo:** Worker de polling processa notificacoes pendentes.

- [ ] `internal/worker/` — polling loop com `time.Ticker`
- [ ] Claim batch com `SKIP LOCKED LIMIT N`
- [ ] Dispatch para EmailProvider (fora da transacao)
- [ ] Complete condicional ao lease token
- [ ] Graceful shutdown via `context.Context`
- [ ] Logs estruturados em cada etapa

**Done when:** Testes unitarios cobrem ciclo sucesso/falha. Teste de integracao cobre recovery de lease expirada.

### Fase 7: Docker final

**Objetivo:** Imagem minima funcional, smoke test automatizado.

- [ ] Dockerfile final com distroless
- [ ] `docker-compose.yml` completo
- [ ] Script de smoke test: sobe container, cria notificacao, worker processa, status = SENT

**Done when:** `docker compose up` + smoke test passa end-to-end.

## Regra de testes

- **Integracao:** apenas fluxo feliz com PostgreSQL real (`testcontainers-go`). Roda no CI, skipa localmente sem banco.
- **Unitario:** todas as variacoes, edge cases e cenarios de erro. Usam fakes em memoria. Rodam sempre (`go test ./...`).

## Fora de escopo (V1)

- Envio real de e-mail ou SMS
- Canal SMS (Evolucao 1)
- Dashboard administrativo
- Templates persistidos
- Filas externas ou broker
- Retry com backoff
- Webhooks de status
- Multiplas contas tecnicas