# ADR-0001. Record architecture decisions

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

Notification Service e um servico novo com escopo deliberadamente pequeno. A maioria do seu valor esta no que ele se recusa a fazer: nao conhece dominios externos, nao envia email real, nao depende de broker. Sem registro escrito, uma recusa e indistinguivel de um descuido, e o proximo contribuidor "conserta".

Varias escolhas de stack neste projeto sao defesaveis nos dois sentidos — UUIDv7 versus ULID, polling worker versus outbox, `pgx` puro versus ORM. Decisoes tomadas sem justificativa registrada sao re-argumentadas cada vez que alguem novo le o codigo. Esse e o anti-padrao Groundhog Day, e e caro precisamente porque os argumentos sao proximos.

## Decision

Toda decisao arquiteturalmente significativa e registrada como um arquivo Markdown numerado em `docs/decisions/`, usando as secoes Context, Decision, Consequences, Compliance, e Notes.

Uma decisao e arquiteturalmente significativa, seguindo o teste de Nygard, quando afeta estrutura, caracteristicas de arquitetura, dependencias, interfaces, ou tecnicas de construcao. Escolhas rotineiras de implementacao nao sao registradas.

Cada ADR apresenta tanto a justificativa tecnica quanto a razao de produto por tras. Uma decisao justificada apenas em termos tecnicos sera reaberta por qualquer um que pese os fatores tecnicos de forma diferente.

ADRs superseded nunca sao deletados ou editados para concordar com o presente. Sao marcados `Superseded by ADR-NNNN`, e o substituto registra por que a situacao mudou. A cadeia e a historia.

## Consequences

**Positive**

- Um sistema de registro. "Por que e assim?" sempre tem um endereco.
- A cadeia de supersession responde "por que nao X?" com evidencia, nao opiniao.
- Novos contribuidores podem ler `docs/decisions/` e entender a forma do sistema antes de ler qualquer codigo.

**Negative**

- Overhead de escrita em toda decisao significativa.
- ADRs podem divergir do codigo. Um ADR que nao descreve mais a realidade e pior que nenhum ADR, porque e confiavel.

## Compliance

Arquivos em `docs/decisions/` sao append-only na pratica: uma mudanca de direcao e um novo ADR que marca o anterior como superseded, nunca uma edicao que reescreve a decisao original. O README e o plan.md linkam para os ADRs.

## Notes

Formato segue a estrutura ADR de Michael Nygard conforme apresentada em *Fundamentals of Software Architecture* (Richards & Ford), capitulo 19.