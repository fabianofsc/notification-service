# ADR-0010. Callback fields for event correlation in future messaging

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** —
- **Superseded by:** —

## Context

A evolucao futura do servico inclui publicacao de eventos de mudanca de status via mensageria (broker). Quando um servico chamador cria uma notificacao, ele precisa de um identificador opaco para correlacionar eventos futuros (`notification.sent`, `notification.failed`) com seu contexto de negocio.

Sem esse identificador, o chamador precisaria manter um mapeamento proprio entre `notification_key` (que ele escolheu) e seu contexto interno. O contrato ja inclui `reference_id` para link com dominio externo, mas eventos precisam de uma tupla `(id, nome)` para roteamento: o broker publica em um topico nomeado, e o consumidor filtra pelo ID.

## Decision

Adicionar dois campos opacos ao contrato de criacao de notificacao: `callback_id` (string) e `callback_name` (string).

- **`callback_id`:** identificador opaco do recurso que originou a notificacao (ex: `order:123`).
- **`callback_name`:** nome opaco do evento ou fluxo que gerou a notificacao (ex: `order_confirmed`).

O servico **nunca interpreta** esses campos. Armazena, ecoa na resposta, e os inclui no fingerprint para que mudancas nos callbacks sejam detectadas como payload diferente (409).

Na evolucao com mensageria, a tupla `(callback_name, callback_id)` define o topico e a chave de particionamento do evento de status.

**Por que dois campos e nao um.** Um unico campo `callback` string obrigaria o chamador a codificar nome e ID em uma string (ex: `order_confirmed:order:123`) e o consumidor a fazer parse. Dois campos separados eliminam convencoes de encoding e permitem que o broker use `callback_name` como topico diretamente.

**Por que opacos.** O servico de notificacao nao conhece dominios externos (regra 1 do AGENTS.md). Interpretar `callback_name` como um tipo de evento violaria essa regra.

**Por que estao no fingerprint.** Se o mesmo `notification_key` for reutilizado com `callback_id` diferente (ex: um retry que mudou o ID de correlacao), o servico deve rejeitar com 409 — e um conflito silencioso de correlacao e pior que um erro explicito.

## Consequences

**Positive**

- Chamadores podem correlacionar eventos futuros sem manter estado externo.
- Broker pode usar `callback_name` como topico e `callback_id` como chave de particionamento.
- Fingerprint detecta mudancas nos callbacks como conflito de payload.

**Negative**

- Dois campos adicionais no contrato, na tabela, e em todas as queries.
- O valor so se justifica quando a mensageria for implementada — e um campo pago adiantado. Mas o custo de adicionar depois (migracao de schema + breaking change no contrato) e maior que o custo de adicionar agora.

## Compliance

Os campos sao `TEXT NOT NULL DEFAULT ''` no banco e `string` (zero value `""`) no dominio. Nao ha validacao alem de serem strings. O fingerprint os inclui na forma canonica `channel|email|subject|body|reference_id|callback_id|callback_name`.

## Notes

Adicionado como refactoring pos-Fase 7, antes do primeiro uso externo do servico. Uma vez que um chamador real dependa do formato de resposta, adicionar campos se torna breaking change ou exige versionamento de API.