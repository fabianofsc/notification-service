# ADR-0006. In-process polling worker with `SKIP LOCKED`

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

Notificacoes sao criadas como `PENDING` e precisam ser processadas por um worker que as reclama, envia, e finaliza. A spec proibe broker externo na V1: o worker deve rodar no mesmo processo Go.

O worker precisa:

- Buscar notificacoes pendentes sem disputa entre multiplas instancias.
- Reclamar notificacoes atomicamente com lease token, para que dois workers nunca processem a mesma.
- Finalizar condicionalmente: um worker sem lease nao pode sobrescrever o resultado de outro.
- Recuperar notificacoes cujo worker morreu (lease expirada).
- Fazer graceful shutdown.

As alternativas consideradas foram: outbox com `due_at`, `LISTEN/NOTIFY`, e polling com `SKIP LOCKED`.

## Decision

Polling worker com `time.Ticker` e `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N`.

```sql
SELECT id, notification_key, channel, recipient, subject, body
FROM notifications
WHERE status = 'PENDING'
   OR (status = 'SENDING' AND lease_until < NOW())
ORDER BY created_at
LIMIT $1
FOR UPDATE SKIP LOCKED
```

O claim seta `status = 'SENDING'`, `lease_token = <uuid>`, `lease_until = NOW() + lease_duration`. A finalizacao condicional:

```sql
UPDATE notifications
SET status = $1, sent_at = $2, failure_reason = $3
WHERE id = $4 AND lease_token = $5
```

Se o `lease_token` nao coincide (outro worker reclamou), a finalizacao e no-op. O worker detecta `rows_affected = 0` e descarta o resultado.

**Por que `time.Ticker` e nao outbox com `due_at`.** O dummy-pay usa outbox porque settlements e webhooks tem delay configurado (`PROCESSING -> APPROVED` apos N segundos). Notificacoes nao tem delay: sao `PENDING` e devem ser processadas o mais rapido possivel. Um ticker simples e suficiente. Outbox adicionaria uma tabela extra (`outbox_work`) sem necessidade.

**Por que nao `LISTEN/NOTIFY`.** Eliminaria polling mas adicionaria complexidade de canal PostgreSQL, reconnect, e backpressure. Para V1 com volume local, o custo de polling a cada 2s e desprezivel. `LISTEN/NOTIFY` pode ser revisitado se a latencia de 2s se tornar inaceitavel.

**Por que nao `pgx` batch com goroutines.** O batch via `SKIP LOCKED LIMIT N` da ao worker N notificacoes por tick. Cada uma e processada em uma goroutine separada, com limite de concorrencia (`MAX_CONCURRENCY`). Isso permite paralelismo sem saturar o pool de conexoes ou o provider fake.

**Parametros configurable:**

| Parametro | Default | Justificativa |
|---|---|---|
| `LEASE_DURATION` | 30s | Provider fake nao tem latencia de rede. 30s e suficiente e evita espera longa em caso de crash. |
| `POLL_INTERVAL` | 2s | Baixa latencia percebida sem sobrecarregar o banco. |
| `BATCH_SIZE` | 10 | Suficiente para throughput local. |
| `MAX_CONCURRENCY` | 5 | Limita goroutines simultaneas. |

**Por que `Clock` injetavel.** O worker usa `clock.Now()` para calcular `lease_until`. Em testes, um clock fake permite controlar o tempo e testar expiracao de lease sem `time.Sleep`.

## Consequences

**Positive**

- Zero dependencias externas. Worker e parte do binario.
- `SKIP LOCKED` garante que N instancias nao disputam as mesmas linhas.
- Lease token garante fencing: so o worker dono da lease finaliza.
- Graceful shutdown com `context.Context`: drena o batch atual antes de parar.

**Negative**

- Latencia de ate `POLL_INTERVAL` (2s) entre criacao e processamento. Aceitavel para V1.
- Polling gera queries regulares ao banco mesmo quando nao ha notificacoes pendentes. Com `SKIP LOCKED` e `LIMIT`, a query e leve.
- Se o worker morrer durante o envio (entre claim e complete), a notificacao fica `SENDING` ate a lease expirar (30s). Nesse periodo, nao e reclamada.

## Compliance

Testes unitarios cobrem: claim com sucesso, claim sem notificacoes pendentes, complete condicional com token correto, complete rejeitado com token errado, recovery de lease expirada. Testes de integracao cobrem o fluxo feliz end-to-end com PostgreSQL real.

O worker usa `internal/clock.Clock` para todo acesso a tempo. Nenhum `time.Now()` ou `time.Sleep()` fora de `internal/clock`.

## Notes

A decisao de in-process worker e compartilhada com dummy-pay (ADR-0008), mas o mecanismo e diferente: dummy-pay usa outbox com `due_at` para trabalho agendado; aqui usamos polling direto na tabela de notificacoes porque nao ha agendamento.