# Notification Service

Servico autonomo de notificacoes, dominio-agnostico, leve para execucao em ambiente local com Docker. V1 simula envio de e-mail; a proxima evolucao adiciona SMS tambem simulado.

- **Por que existe** — [docs/spec-v1.md](docs/spec-v1.md)
- **Por que foi construido assim** — [docs/decisions/](docs/decisions/)
- **Como foi construido** — [docs/plan-v1.md](docs/plan-v1.md)

> **Status: V1 implementada.** 130 testes passam, Docker compose sobe com smoke test end-to-end (create → worker → SENT). Veja [o roadmap](docs/plan-v1.md#fases-de-implementacao) para o status por etapa.

---

## O que faz

Recebe solicitacoes de notificacao com mensagem ja renderizada, deduplica por chave idempotente, e entrega via canais simulados (email na V1, SMS depois). Nao conhece pedidos, pagamentos, clientes ou qualquer outro dominio externo.

## Stack

| Concern | Choice |
|---|---|
| Linguagem | Go |
| Arquitetura | Hexagonal leve |
| Banco de dados | PostgreSQL via `pgx/v5`, SQL escrito a mao |
| Identificadores | UUIDv7, prefixado na borda HTTP (`ntf_`, `dly_`) |
| Migrations | `golang-migrate` com `embed` |
| Logs | `slog` (stdlib) |
| Autenticacao | Basic Auth, conta tecnica por ambiente |
| Worker | In-process com `time.Ticker` + `SKIP LOCKED` |
| Docker | Multi-stage -> `distroless/static` |

Decisoes completas em [docs/decisions/](docs/decisions/).

## API

### POST /v1/notifications

Cria uma notificacao. Idempotente via `Idempotency-Key`.

```
Authorization: Basic <credenciais>
Idempotency-Key: order-confirmed:123:attempt-456
```

```json
{
  "notification_key": "order-confirmed:123:attempt-456",
  "channel": "EMAIL",
  "recipient": {
    "email": "cliente@example.com"
  },
  "subject": "Pedido confirmado",
  "body": "Seu pedido 123 foi confirmado.",
  "reference_id": "order:123",
  "callback_id": "order:123",
  "callback_name": "order_confirmed"
}
```

**Resposta — `202 Accepted`**

```json
{
  "notification_id": "ntf_0199a1f4-3c82-7d19-b4e6-2f8a91c05d3b",
  "notification_key": "order-confirmed:123:attempt-456",
  "channel": "EMAIL",
  "status": "PENDING",
  "reference_id": "order:123",
  "callback_id": "order:123",
  "callback_name": "order_confirmed",
  "created_at": "2026-08-11T12:00:00Z"
}
```

**Idempotencia:**
- Mesma chave + mesmo payload -> `202` com a notificacao existente
- Mesma chave + payload diferente -> `409 Conflict`

### GET /v1/notifications/{notification_id}

Retorna estado atual, tentativas, timestamps e motivo de falha.

### GET /health

```json
{"status":"ok"}
```

Retorna sucesso quando o servico e o PostgreSQL estao acessiveis.

## Configuracao

Tudo por variaveis de ambiente. Nao ha arquivo de configuracao com credenciais.

| Variavel | Required | Default | Descricao |
|---|---|---|---|
| `DATABASE_URL` | yes | — | PostgreSQL DSN |
| `BASIC_AUTH_USERNAME` | no | `notification` | Basic Auth username |
| `BASIC_AUTH_PASSWORD` | no | `notification` | Basic Auth password |
| `PORT` | no | `8080` | Porta HTTP |
| `LEASE_DURATION` | no | `30s` | Duracao da lease |
| `POLL_INTERVAL` | no | `2s` | Intervalo de poll |
| `BATCH_SIZE` | no | `10` | Tamanho do batch |
| `MAX_CONCURRENCY` | no | `5` | Concorrencia maxima |

Copie `.env.example` para `.env` e preencha os valores reais.

## Rodando localmente

```sh
cp .env.example .env
docker compose up -d      # PostgreSQL + servico
curl http://localhost:8080/health
# {"status":"ok"}
```

## Testes

```sh
go test ./...              # unitarios (sempre rodam)
go test ./... -race -count=5
```

Testes de integracao usam PostgreSQL real (`testcontainers-go`). Sem banco, eles skipam.

## Escopo

**V1:** notificacoes por EMAIL (fake), deduplicacao, worker in-process, lease-based concurrency.

**Fora de escopo:** envio real de e-mail/SMS, canal SMS, templates persistidos, filas externas, webhooks, broker, dashboard.