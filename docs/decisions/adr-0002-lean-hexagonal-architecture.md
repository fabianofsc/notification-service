# ADR-0002. Lean hexagonal architecture with `internal/` layout

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

O servico precisa ser dominio-agnostico e autonomo. V1 expoe HTTP sync; evolucoes futuras adicionam entrada assincrona (mensageria). O dominio nao pode depender de nenhum mecanismo de transporte.

O layout precisa isolar o dominio de frameworks e bibliotecas de infra, e precisa suportar substituicao de adapters sem tocar no nucleo.

As opcoes consideradas foram: hexagonal classico com `pkg/` exportavel, flat com tudo em um pacote, e hexagonal leve com `internal/`.

## Decision

Hexagonal leve: `cmd/server/main.go` e o unico ponto de wiring. O dominio (`internal/domain`) e os use cases (`internal/app`) declaram ports (interfaces). Adapters (`internal/http`, `internal/postgres`, `internal/email`, `internal/worker`) implementam esses ports.

Layout:

```
cmd/server/main.go
internal/
  domain/       # tipos puros, zero imports de infra
  app/          # ports (interfaces) + use cases
  http/         # handler, router, auth, idcodec, dto
  postgres/     # repository adapter, migrations
  email/        # EmailProvider fake adapter
  worker/       # polling worker
  config/       # env parsing
```

**Por que `internal/` sem `pkg/`.** O servico e um binario unico na V1. Nao ha contrato publico alem do HTTP. Se SDKS ou bibliotecas compartilhadas forem necessarios no futuro, um novo ADR decide qual pacote exportar. Exportar prematuramente cria superficie de API publica sem consumidores, e mudar API publica depois e breaking change.

**Por que hexagonal e nao flat.** A entrada futura assincrona (mensageria) e um adapter diferente no mesmo port de entrada. Se o dominio conhecesse HTTP, trocar para mensageria exigiria reescrever logica de negocio. Com hexagonal, e um novo adapter satisfazendo o mesmo port.

**Por que `app/` e nao `usecase/`.** O pacote `app/` contem tanto os ports quanto os use cases. Mantem o contrato (interface) e a orquestracao (use case) no mesmo pacote, visivel para todos os adapters, mas sem expor implementacao de dominio.

## Consequences

**Positive**

- Dominio nao conhece HTTP, PostgreSQL, ou email. Testavel em memoria com fakes.
- Adapter assincrono futuro (mensageria) e drop-in no mesmo port de entrada.
- Layout previsivel: toda decisao de "onde isso vai?" tem uma resposta.
- `internal/` previne imports acidentais de outros servicos.

**Negative**

- Mais arquivos que um flat layout. Overhead perceptivel para um servico com 2 endpoints.
- Wrapper boilerplate: cada adapter implementa uma interface declarada em `app/`.
- Nao ha codigo reutilizavel exportado — se outro servico precisar de um client HTTP para este, ele escreve o proprio.

## Compliance

`internal/domain` importa apenas a biblioteca padrao, `uuid`, e `internal/clock`. Nao importa `internal/http`, `internal/postgres`, `internal/email`, ou `internal/worker`. Violacao e detectavel por `go vet` e pela regra no AGENTS.md.

## Notes

Padrao estabelecido no dummy-pay (ADR-0003) e replicado aqui. A unica diferenca e que dummy-pay usa `internal/payment` como nome do dominio; aqui usamos `internal/domain` para generalidade.