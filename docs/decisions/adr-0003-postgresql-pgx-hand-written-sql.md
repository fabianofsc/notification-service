# ADR-0003. PostgreSQL via `pgx/v5`, hand-written SQL, `golang-migrate`

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

O servico depende de PostgreSQL para idempotencia, claim atomico de lease, e finalizacao condicional de delivery. Precisamos de um driver PostgreSQL e uma ferramenta de migracao.

As garantias criticas sao:

- `INSERT ... ON CONFLICT (notification_key)` para idempotencia entre processos.
- `SELECT ... FOR UPDATE SKIP LOCKED LIMIT N` para claim atomico sem disputa entre workers.
- `UPDATE ... WHERE lease_token = $1` para finalizacao condicional — um worker sem lease nao pode sobrescrever o resultado de outro.

Um ORM esconderia exatamente o SQL que precisa ser revisavel. A alternativa e SQL escrito a mao com `pgx/v5`, o driver PostgreSQL mais performatico e idiomatico para Go.

Para migracoes, `golang-migrate/migrate` e a ferramenta estabelecida no ecossistema (usada pelo dummy-pay), com suporte a `embed` para embutir migrations no binario.

## Decision

- **Driver:** `github.com/jackc/pgx/v5` com `pgxpool` para connection pooling.
- **SQL:** Escrito a mao, sem query builder ou ORM.
- **Migracoes:** `github.com/golang-migrate/migrate/v4` com driver `pgx/v5`. Arquivos SQL embedados via `embed.FS`, aplicados no startup antes do HTTP server subir.
- **Schema:** Migration files numerados sequencialmente em `internal/postgres/migrations/`.

**Por que `pgx` e nao `database/sql` com `lib/pq`.** `pgx` e nativo PostgreSQL, oferece suporte completo a tipos PostgreSQL (incluindo `uuid` nativo, `jsonb`, `timestamptz`), e connection pooling via `pgxpool`. `lib/pq` esta em modo de manutencao.

**Por que SQL a mao e nao ORM.** As queries de claim atomico (`FOR UPDATE SKIP LOCKED`) e complete condicional (`WHERE lease_token = $1`) sao o nucleo da correcao do worker. Um ORM geraria SQL opaco e esconderia race conditions. SQL a mao e revisavel e testavel.

**Por que `golang-migrate` e nao `goose`.** Ambos sao maduros. `golang-migrate` e mais simples (so SQL, sem Go migrations), e ja e usado no dummy-pay. Consistencia com o ecossistema pesa mais que features extras.

**Por que `embed` e nao arquivos externos.** Schema e binario viajam juntos. Nao ha risco de versao errada da migration. Clone e run nao precisa de ferramenta extra.

## Consequences

**Positive**

- SQL revisavel. As queries que garantem idempotencia e fencing de lease sao visiveis e testaveis.
- Nenhuma camada de abstracao entre o codigo e o banco. Performance previsivel.
- Migrations aplicadas automaticamente no boot. Impossivel esquecer.

**Negative**

- Mais codigo boilerplate para scan de linhas (mapeamento manual de colunas).
- Mudancas de schema exigem escrever SQL bruto; sem migracoes自动 geradas por ORM.
- `pgx` e especifico do PostgreSQL. Trocar de banco exigiria reescrever toda a camada de persistencia — mas nao ha plano de trocar de banco.

## Compliance

Todo SQL vive em `internal/postgres/`. Nenhum outro pacote escreve SQL ou importa `pgx`. Migrations estao em `internal/postgres/migrations/` e sao embedadas no binario. O AGENTS.md declara a regra.

## Notes

Decisao equivalente ao dummy-pay ADR-0004, com a diferenca de que dummy-pay usa `goose` enquanto aqui usamos `golang-migrate`. A troca e justificada pela menor superficie: sem Go migrations, apenas SQL.