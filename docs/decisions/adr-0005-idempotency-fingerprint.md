# ADR-0005. Idempotency via unique constraint with payload fingerprint

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

A API de criacao de notificacao e idempotente: a mesma `notification_key` com o mesmo payload deve retornar a notificacao existente. A mesma chave com payload diferente deve retornar `409 Conflict`.

A idempotencia precisa funcionar entre processos (o servico pode ter multiplas instancias), entao locking in-process e insuficiente. A garantia precisa vir do banco.

Precisamos decidir como detectar "mesmo payload" vs "payload diferente" de forma deterministica e eficiente.

## Decision

Combina unique constraint no banco com hash de fingerprint.

1. **Unique constraint** em `notification_key` na tabela `notifications`. A corrida entre requests concorrentes e resolvida pelo PostgreSQL: `INSERT ... ON CONFLICT (notification_key) DO NOTHING`. O vencedor escreve; os perdedores detectam o conflito e leem a linha existente.

2. **Payload fingerprint** como coluna `payload_fingerprint TEXT NOT NULL`. O fingerprint e um hash SHA-256 dos campos canonicos normalizados, calculado no dominio antes de persistir.

3. **Normalizacao canonica** antes do hash: `channel|recipient_email|subject|body|reference_id`. Campos sao lowercased onde aplicavel, whitespace trimmed. Para recipient JSONB no futuro SMS, o phone_number e normalizado (E.164) e incluido no fingerprint.

4. **Logica de conflito:** apos `ON CONFLICT`, le a linha existente. Se `payload_fingerprint` coincide -> retorna a existente (202). Se diferente -> retorna 409 Conflict.

**Por que hash e nao comparacao campo a campo.** Comparar 5 campos em Go ou SQL e mais codigo e mais lento. Um hash de 64 chars e uma comparacao de igualdade. SHA-256 e deterministico e a probabilidade de colisao e astronomicamente baixa para este volume.

**Por que SHA-256 e nao MD5 ou SHA-1.** SHA-256 e o padrao moderno. Nao ha preocupacao com ataque de colisao aqui (nao e seguranca, e deduplicacao), mas usar um hash deprecated levantaria questionamentos desnecessarios em review.

**Por que normalizacao e nao hash do JSON bruto.** JSON bruto e sensivel a whitespace e ordem de campos. Dois payloads semanticamente identicos mas com formatacao diferente produziriam hashes diferentes e um 409 incorreto. A normalizacao garante equivalencia semantica.

## Consequences

**Positive**

- Garantia de idempotencia no banco, nao em memoria. Funciona com N instancias.
- Fingerprint e uma comparacao O(1). Nao escala com numero de campos.
- Normalizacao evita falsos 409 por formatacao.

**Negative**

- Coluna extra no schema.
- Logica de normalizacao precisa ser mantida em sync com os campos do payload. Se um campo novo for adicionado ao request, a normalizacao precisa ser atualizada.
- SHA-256 produz 64 chars hex — visivel em dumps de banco, mas nao e dado sensivel.

## Compliance

Testes de idempotencia cobrem: replay igual retorna existente, payload diferente retorna 409, e requests concorrentes com a mesma chave resolvem corretamente. O fingerprint e calculado no dominio (`internal/domain`), nao no adapter HTTP.

## Notes

A abordagem de fingerprint + unique constraint e uma variacao do padrao usado no dummy-pay ADR-0007. Dummy-pay usa `IN_FLIGHT`/`COMPLETED` para detectar corrida; aqui usamos fingerprint para detectar conflito de payload. Sao problemas diferentes resolvidos com o mesmo mecanismo de base: unique constraint no banco.