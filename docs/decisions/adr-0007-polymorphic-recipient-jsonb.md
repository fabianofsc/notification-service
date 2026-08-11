# ADR-0007. Polymorphic recipient as JSONB with search index

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** -
- **Superseded by:** -

## Context

V1 suporta apenas canal EMAIL com recipient `{"email": "cliente@example.com"}`. A Evolucao 1 adiciona SMS com `{"phone_number": "+5511999999999"}`. Futuros canais (push notification, WhatsApp) terao seus proprios formatos de recipient.

Precisamos de um modelo de dados que:

- Valide o recipient conforme o channel (EMAIL requer email valido, SMS requer phone number valido).
- Permita adicionar novos canais sem migracao de schema.
- Permita busca eficiente por destinatario (ex: "encontre notificacoes para este email").

As alternativas: colunas separadas nullable, tabela polimorfica, e JSONB.

## Decision

Coluna `recipient JSONB NOT NULL` + coluna `recipient_search VARCHAR INDEX` com valor normalizado para busca.

```sql
recipient        JSONB NOT NULL,   -- {"email": "cliente@example.com"} ou {"phone_number": "+5511999999999"}
recipient_search VARCHAR NOT NULL, -- "cliente@example.com" ou "+5511999999999"
CREATE INDEX idx_notifications_recipient_search ON notifications(recipient_search);
```

**Validacao no dominio.** O domain type `Recipient` valida o JSON contra o channel. Se channel e EMAIL, `recipient.email` e obrigatorio e deve ser email valido. Se SMS, `recipient.phone_number` e obrigatorio e deve ser E.164 valido. Campos desconhecidos sao rejeitados.

**`recipient_search`.** Extraido do JSONB no momento do insert: email lowercased e trimmed, ou phone_number normalizado E.164. Indexado para queries de busca. Nao e exposto na API — e detalhe de implementacao.

**Por que JSONB e nao colunas separadas.** Colunas separadas (`recipient_email VARCHAR NULL`, `recipient_phone VARCHAR NULL`) exigiriam CHECK constraint para garantir que so uma esta preenchida, e cada novo canal exigiria `ALTER TABLE ADD COLUMN`. JSONB aceita qualquer formato sem migracao.

**Por que nao tabela polimorfica.** Uma tabela `recipients` com `recipient_type` e colunas especificas e overengineering para 2 canais. Adiciona JOIN em toda query e complexidade de FK.

**Por que `recipient_search` e nao index GIN no JSONB.** Indice GIN em JSONB funciona para o operador `@>` mas e mais pesado que um B-tree em `VARCHAR`. O `recipient_search` e um valor deterministico e indexado de forma leve.

## Consequences

**Positive**

- Adicionar SMS na Evolucao 1 nao requer migracao de schema — so adicionar validacao no dominio.
- Busca por destinatario e eficiente via indice B-tree.
- Validacao no dominio garante que JSONB malformado nunca chega ao banco.

**Negative**

- JSONB e opaco para o PostgreSQL — nao ha FK ou constraint no conteudo do JSON.
- `recipient_search` e uma coluna derivada que precisa ser mantida em sync com `recipient`. Se um bug popular `recipient_search` incorretamente, buscas retornam resultados errados.
- O JSONB armazena o email em plaintext — aceitavel porque o email ja esta no corpo da notificacao e o servico e interno.

## Compliance

Testes de dominio validam recipient para EMAIL (email obrigatorio, formato valido, rejeita campos desconhecidos) e preparam o ground para SMS. O repository extrai `recipient_search` do recipient validado — nunca faz parse proprio do JSONB.

## Notes

O modelo de recipient polimorfico e uma decisao antecipatoria: so EMAIL existe na V1, mas o schema ja suporta SMS sem migracao. Isso e uma excecao consciente a regra de "nao se prepare para o futuro" — o custo de fazer JSONB agora vs `VARCHAR` e depois migrar e assimetrico: JSONB agora custa quase nada; migrar depois e uma operacao de schema em producao.