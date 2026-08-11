# ADR-0004. UUIDv7 identifiers, prefixed at the HTTP boundary

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

O servico gera dois tipos de identificadores publicos: `notification_id` e `delivery_id`. Eles sao opacos para os chamadores e devem ser estaveis pela vida do registro.

Os candidatos eram inteiros sequenciais, UUIDv4, UUIDv7, e ULID.

Uma restricao nao obvia: o ID deve existir *antes* da linha ser escrita, porque e gerado no dominio e passado para o repository. Isso descarta qualquer coisa que o banco atribui no insert.

## Decision

UUIDv7 (RFC 9562), gerado na aplicacao via `github.com/google/uuid`, armazenado em colunas PostgreSQL nativas `uuid`.

Na borda HTTP — e somente la — os identificadores sao renderizados com prefixo de tipo: `ntf_` para notification, `dly_` para delivery. O prefixo e apresentacao. Nunca e armazenado e nunca faz parte do valor da coluna.

A implementacao segue o padrao estabelecido no dummy-pay (ADR-0006): um `UUIDv7Generator` em `internal/clock/id.go`, um codec `encodeID`/`decodeID` em `internal/http/idcodec.go`, e tipos `uuid.UUID` em todo o dominio e banco.

**Por que nao inteiros sequenciais.** Revelam volume, sao enumeraveis, e sao atribuidos pelo banco no insert — incompativel com gerar o ID no dominio antes de persistir.

**Por que nao UUIDv4.** Chaves aleatorias dispersam insercoes B-tree pelo indice inteiro, causando page splits. UUIDv4 tambem nao carrega ordenacao, entao qualquer query ordenada por tempo precisa de um indice separado.

**Por que UUIDv7 e nao ULID.** Tecnicamente quase equivalentes: ambos sao 128 bits com timestamp de milissegundo nos bits altos. O desempate e integracao: UUIDv7 e um padrao IETF, PostgreSQL tem tipo nativo `uuid` (16 bytes), e `google/uuid` gera nativamente. ULID precisaria de `char(26)` ou bytes crus enfiados em uma coluna `uuid` com nibbles de versao/variant invalidos.

**Por que prefixo.** Um identificador prefixado e auto-descritivo em uma linha de log, e torna obvio quando um delivery ID foi passado onde um notification ID e esperado — um erro que seria invisivel de outra forma, ja que ambos sao UUIDs.

## Consequences

**Positive**

- Insercoes ordenadas por tempo mantem localidade de indice boa.
- `ORDER BY id` e cronologico.
- Colunas `uuid` nativas: 16 bytes, sem cast, sem decisoes de encoding.
- Confusao de tipo entre identificadores e pega na borda em vez de produzir um 404 misterioso.

**Negative**

- Longo em logs e URLs. E o custo concreto de nao escolher ULID.
- Logica de encode/decode na borda, mais testes para isso.
- UUIDv7 embute um timestamp de criacao — um identificador revela quando o registro foi criado. Aceitavel para este servico.

## Compliance

Um teste de borda assere que um UUID bem-formado com o prefixo errado (ex: `dly_` onde `ntf_` e esperado) e rejeitado. Tipos de dominio e storage sao `uuid.UUID`, nunca `string`, entao um valor prefixado nao pode alcancar o banco.

## Notes

Replica o padrao do dummy-pay ADR-0006. Os prefixos sao diferentes (`ntf_`/`dly_` vs `pay_`/`txn_`/etc.), mas o mecanismo e identico.