# AGENTS.md — Notification Service

Servico autonomo de notificacoes, dominio-agnostico. Go, PostgreSQL proprio, dois endpoints sob `/v1`, worker in-process sem broker. Nao conhece pedidos, pagamentos ou clientes.

Leia estes antes de qualquer coisa. Sao o contrato, nao contexto.

@README.md
@docs/spec-v1.md
@docs/plan-v1.md
@docs/decisions/

ADRs individuais em `docs/decisions/adr-NNNN-*.md`. Abra o ADR que a tarefa toca; nao raciocine sobre uma decisao a partir do resumo de uma linha no indice.

---

## Regras nao-negociaveis

Nao sao preferencias de estilo. Violar uma e um defeito, e cada uma rastreia um ADR que explica o custo da alternativa.

1. **O servico nunca conhece dominios externos.** Nada de customer_id, order_id, payment_id, ou qualquer FK para outro servico. `reference_id` e string opaca. O servico recebe mensagem ja renderizada. (Spec SS2)
2. **Nenhum sistema externo.** Nao ha broker, cache, provedor de email real, ou API externa na V1. A suite de testes deve passar sem rede externa.
3. **`internal/domain` nao importa nada de infra.** Nao importa `internal/http`, `internal/postgres`, `internal/email`. O dominio declara seus ports; adapters implementam. (ADR-0002)
4. **Nao use `time.Now()` ou `time.Sleep()` fora de `internal/clock`.** Tempo e injetado via interface `Clock`. Nenhum teste dorme. (ADR-0006)
5. **Configuracao vem apenas do ambiente.** Nenhum arquivo de config e lido. `.env.example` tem placeholders, nunca valores reais.
6. **SQL vive apenas em `internal/postgres`.** (ADR-0003)
7. **Nunca logar corpo completo de e-mail ou dados do recipient.** Logs estruturados mencionam notification_id e delivery_key, nunca subject, body, email ou phone_number.
8. **Para reverter uma decisao, escreva um novo ADR** marcando o anterior como superseded. Nunca edite um registro superseded para concordar com o presente. (ADR-0001)

---

## Workflow

**Teste primeiro, sempre.** Escreva o teste, rode, veja falhar pela razao certa, depois implemente. Um teste que passou de primeira nao prova nada ate voce ver ele falhar.

**Siga a ordem do plano.** `docs/plan-v1.md` sequencia o trabalho para que cada fase seja independentemente verificavel. A Fase 3 (Persistencia) e a de maior risco — se os testes de claim atomico nao forem convincentes, pare e corrija o design antes de construir em cima.

**Marque o checkbox no commit que o conquista.** Um checkbox em `docs/plan.md#fases-de-implementacao` e marcado quando a condicao **Done when** daquela fase foi *observada*, nao quando o codigo foi escrito. Nunca marque adiantado.

**Reporte honestamente.** Se testes falharem, diga e mostre a saida. Se uma fase foi pulada, diga qual e por que. Nunca afirme que uma fase esta pronta sem rodar sua verificacao.

---

## Comandos

```sh
make db-up               # sobe PostgreSQL via docker compose
make test                # testes unitarios; nao precisam de banco
make test-integration    # testes de integracao; requer banco
make lint                # vet + verificacoes estaticas
make run                 # roda o servico contra o banco do compose
go test ./...            # tudo; integracao skipa sem banco
go test ./... -race -count=5   # rode ao final de cada fase
```

Testes de integracao skipam quando nao ha banco acessivel, entao `go test ./...` funciona em qualquer lugar. Em CI um skip e **falha** — nunca confie apenas em um green local como evidencia de garantia que depende de banco.

---

## Convencoes

**Layout** — ver spec SStack e plan.md. `cmd/server/main.go` e o unico lugar onde adapters sao construidos.

**Erros** — erros de dominio sao sentinel values em `internal/domain`, mapeados para HTTP codes apenas em `internal/http`. Wrap com `%w`. Um body `500` nunca contem o erro subjacente; ele e logado.

**Identificadores** — `uuid.UUID` no dominio e no banco, nunca `string`. O prefixo `ntf_`/`dly_` e aplicado e removido apenas na borda HTTP. Um UUID bem-formado com o prefixo errado e `404`, nao um lookup. (ADR-0004)

**Testes** — table-driven, na stdlib `testing`. `testify/require` para controle de fluxo; `cmp.Diff` reportado como `(-want +got)` para comparar structs. Nunca `assert` — apenas `require`. Nunca `testify/mock` ou `testify/suite`: fakes sao escritos a mao, e `t.Cleanup` cobre teardown. Testes de integracao ganham schema proprio e nao compartilham estado.

> **Nunca use `require.Equal` em struct contendo timestamp.** Cai em `reflect.DeepEqual`, que compara monotonic reading e location pointer, entao um valor que passou pelo PostgreSQL falha contra o original mesmo quando os instants sao iguais. Use `cmp.Diff`, que respeita `time.Time.Equal`.

**Dependencias** — `pgx`, `golang-migrate`, `google/uuid`, `testify`, `go-cmp`, `testcontainers-go`. Adicionar outra e uma decisao de arquitetura e precisa de um ADR.

**Commits** — subject imperativo ate 72 caracteres, prefixado por area (`feat:`, `fix:`, `docs:`, `test:`, `refactor:`). O corpo explica por que, nao o que. Nao commite ou de push a menos que solicitado.

---

## Fora de escopo

Nao construa estes, e nao "se prepare" para eles. Cada ausencia e uma decisao registrada no spec e nos ADRs:

Envio real de e-mail ou SMS · canal SMS (Evolucao 1, futura) · dashboard administrativo · templates persistidos · filas externas ou broker · retry com backoff · webhooks de status · multiplas contas tecnicas · push notification · integracao com qualquer outro servico.

Se uma tarefa parece precisar de um destes, pare e pergunte. E mais provavel que a tarefa esteja errada do que o escopo.

---

## Estado atual

Fases 1-7 implementadas: scaffold, dominio, persistencia, HTTP API, EmailProvider fake, worker, e Docker final. 130 testes passam, 595 com `-race -count=5`. Docker compose sobe com `/health` funcional e smoke test end-to-end passa (create -> worker -> SENT). Veja [o roadmap](docs/plan-v1.md#fases-de-implementacao) para o status por etapa.