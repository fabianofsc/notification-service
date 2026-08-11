# ADR-0008. Basic Auth with single technical account per environment

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

O servico expoe endpoints HTTP que serao chamados por outros servicos internos (nexus-shopping, dummy-pay, etc.). Precisamos de autenticacao, mas o escopo e minimo: uma conta tecnica por ambiente, sem gerenciamento de usuarios, sem OAuth, sem JWT.

A spec define que todas as APIs usam `Authorization: Basic <credenciais>`. As credenciais sao configuradas por variaveis de ambiente.

## Decision

HTTP Basic Authentication com uma unica conta tecnica por ambiente.

- **Credenciais:** `BASIC_AUTH_USERNAME` e `BASIC_AUTH_PASSWORD` como env vars.
- **Middleware:** aplicado em todas as rotas sob `/v1`. `/health` nao requer auth.
- **Defaults locais:** `notification` / `notification` para desenvolvimento. Em producao, as env vars sao obrigatorias e o servico nao sobe sem elas.
- **Header:** `Authorization: Basic base64(username:password)`. Missing ou invalido -> `401 Unauthorized`.

**Por que Basic Auth e nao API key.** Basic Auth e um padrao HTTP universal. Todo client HTTP suporta nativamente. API key em header customizado (ex: `X-API-Key`) e igualmente simples mas menos padronizado. Para comunicacao interna entre servicos, Basic Auth e suficiente.

**Por que nao OAuth/JWT.** O servico nao gerencia usuarios nem sessoes. Nao ha necessidade de tokens com expiry, scopes, ou refresh. Introduzir OAuth para uma conta tecnica seria overengineering.

**Por que nao mTLS.** mTLS exige gerenciamento de certificados — uma infraestrutura que nao existe no ambiente local. Pode ser adicionado no futuro sem mudar o modelo de autenticacao (Basic Auth sobre mTLS e comum).

**Por que defaults locais.** Facilita desenvolvimento: `docker compose up` funciona sem configurar env vars. Em producao, as credenciais serao fornecidas pelo orquestrador (Kubernetes secrets, etc.).

## Consequences

**Positive**

- Simples de implementar e testar.
- Middleware reutilizavel padrao. Nenhuma biblioteca externa necessaria.
- Compatível com qualquer client HTTP.

**Negative**

- Credenciais em Base64 trafegam em plaintext (sem HTTPS). Aceitavel para comunicacao interna em localhost/VPC. Em producao com rede externa, HTTPS e obrigatorio — mas e preocupacao da infra, nao do servico.
- Rotacao de credenciais exige restart do servico. Aceitavel para V1.
- Sem audit log de quem chamou (todas as chamadas sao da mesma conta).

## Compliance

Todo endpoint sob `/v1` requer auth. Testes HTTP (`httptest`) cobrem: sem header -> 401, credenciais invalidas -> 401, credenciais validas -> acesso permitido. `/health` e publico e nao requer auth.

## Notes

Decisao analoga ao dummy-pay, que tambem usa Basic Auth com conta tecnica unica (`key_id`/`key_secret`). A diferenca e que dummy-pay usa nomes `ACCOUNT_KEY_ID`/`ACCOUNT_KEY_SECRET` para as env vars; aqui usamos `BASIC_AUTH_USERNAME`/`BASIC_AUTH_PASSWORD` para clareza.