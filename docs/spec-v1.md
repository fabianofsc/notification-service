# Spec: Notification Service em Go

**Status:** Aprovada
**Objetivo:** Criar um microservico autonomo de notificacoes, leve para execucao em ambiente local com Docker. A V1 simula envio de e-mail; a proxima evolucao adiciona SMS tambem simulado.

## Stack

- Go
- HTTP/JSON
- PostgreSQL
- Migrations SQL
- Docker com imagem final minima
- `net/http` e biblioteca PostgreSQL leve, como `pgx`
- Logs estruturados via `slog`

O servico deve ter binario unico, boot rapido e nao depender de broker, cache ou provedor externo na V1.

## Limites do servico

O servico e dono de:

- solicitacoes de notificacao;
- deduplicacao;
- ciclo de vida de entrega;
- tentativas de envio;
- configuracao de canais;
- adaptadores de envio.

O servico nao conhece pedidos, pagamentos, clientes ou qualquer outro dominio externo. Recebe apenas uma referencia opaca e uma mensagem ja renderizada.

Cada servico deve ter banco e usuario PostgreSQL proprios. Em ambiente local, varios bancos logicos podem compartilhar uma unica instancia PostgreSQL, sem acesso cruzado.

## Linguagem ubiqua

- **Notification:** solicitacao persistida para comunicar uma mensagem.
- **Notification Key:** chave idempotente fornecida pelo cliente.
- **Channel:** meio de entrega; inicialmente `EMAIL`, depois `SMS`.
- **Recipient:** destino da notificacao.
- **Delivery:** tentativa concreta de envio por um canal.
- **Provider:** adapter responsavel por enviar, inicialmente fake.
- **Lease:** posse temporaria de uma entrega por um worker.
- **Terminal status:** estado final `SENT` ou `FAILED`.

## Estados

```text
PENDING -> SENDING -> SENT
                   -> FAILED

- PENDING: notificacao aceita e aguardando processamento.
- SENDING: worker possui lease valida e esta enviando.
- SENT: entrega concluida.
- FAILED: tentativa terminou com falha.
- Um worker que perdeu a lease nao pode alterar o resultado de outro worker.
```

## API V1

Todas as APIs internas usam:

Authorization: Basic <credencial da conta tecnica>

Existe uma conta tecnica por ambiente, configurada por variaveis de ambiente.

### Criar notificacao

POST /v1/notifications
Idempotency-Key: <chave nao vazia>

Corpo:

```json
{
  "notification_key": "order-confirmed:123:attempt-456",
  "channel": "EMAIL",
  "recipient": {
    "email": "cliente@example.com"
  },
  "subject": "Pedido confirmado",
  "body": "Seu pedido 123 foi confirmado.",
  "reference_id": "order:123"
}
```

Regras:

- notification_key e obrigatoria e unica.
- channel aceita somente EMAIL na V1.
- recipient.email deve ser valido.
- subject e body sao obrigatorios para e-mail.
- reference_id e opaco.
- A mesma chave com o mesmo payload devolve a notificacao existente.
- A mesma chave com payload diferente retorna 409 Conflict.
- O servico devolve 202 Accepted com a notificacao em PENDING.

Resposta:

```json
{
  "notification_id": "ntf_01...",
  "notification_key": "order-confirmed:123:attempt-456",
  "channel": "EMAIL",
  "status": "PENDING",
  "reference_id": "order:123",
  "created_at": "2026-08-11T12:00:00Z"
}
```

### Consultar notificacao

GET /v1/notifications/{notification_id}

Retorna estado atual, quantidade de tentativas, timestamps e motivo de falha quando existir.

### Health

GET /health

Retorna sucesso somente quando o servico e o PostgreSQL estiverem acessiveis.

## Processamento

1. A API valida e persiste a notificacao como PENDING.
2. Um worker interno busca notificacoes pendentes.
3. O worker reclama a notificacao atomicamente com lease e token de posse.
4. O adapter do canal e chamado fora da transacao do banco.
5. O resultado e persistido de modo condicional ao token da lease.
6. Falha de processo ou lease expirada permite nova reclamacao.

Nao usar broker, fila externa ou scheduler separado na V1. O worker roda no mesmo processo Go.

## Provider fake de e-mail

Criar uma porta interna equivalente a:

EmailProvider.send(recipient, subject, body, deliveryKey) -> DeliveryResult

O adapter fake:

- nao chama servico externo;
- nao envia e-mail real;
- registra log estruturado sem expor conteudo sensivel;
- retorna sucesso por padrao;
- permite falha deterministica por configuracao ou token de cenario para testes;
- preserva deliveryKey para rastreabilidade e deduplicacao.

## Persistencia minima

Tabelas sugeridas:

- notifications
    - id
    - notification_key
    - payload_fingerprint
    - channel
    - recipient
    - subject
    - body
    - reference_id
    - status
    - lease_token
    - lease_until
    - attempt_count
    - failure_reason
    - created_at
    - sent_at
    - updated_at

- notification_deliveries
    - id
    - notification_id
    - delivery_key
    - status
    - attempt_number
    - provider_response
    - failure_reason
    - created_at
    - completed_at

Criar unicidade para notification_key e para delivery_key.

## Seguranca

- Nunca registrar corpo completo, e-mail ou telefone sem necessidade operacional.
- Nunca armazenar credenciais no repositorio.
- Configurar credenciais e URL do PostgreSQL por ambiente.
- Nao aceitar HTML, anexos ou dados sensiveis na V1.
- Nao implementar envio real de e-mail ou SMS na V1.

## Testes obrigatorios

- Dominio: transicoes de estado e fencing de lease.
- Idempotencia: replay igual, conflito de payload e concorrencia.
- Persistencia PostgreSQL: unicidade, claim atomico e finalizacao condicional.
- HTTP: autenticacao, validacao, 202, replay e 409.
- Worker: e-mail fake com sucesso e falha.
- Recuperacao: lease expirada pode ser reclamada.
- Container: servico sobe localmente com PostgreSQL e responde health.

## Evolucao 1: SMS fake

Adicionar SMS como novo channel, sem alterar o ciclo de vida ou a API de notificacao.

Novo formato de recipient:

```json
{
  "phone_number": "+5511999999999"
}
```

Criar SmsProvider fake com o mesmo contrato de entrega do e-mail. Nenhuma integracao real e permitida nessa fase.

## Evolucoes futuras

- Providers reais de e-mail e SMS.
- Templates e renderizacao no proprio servico.
- Retry com backoff, limite de tentativas e dead-letter queue.
- Webhooks de status de entrega dos providers.
- Eventos via broker.
- Multiplas contas tecnicas, rotacao de credenciais e rate limiting.
- Push notification e outros canais.

## Fora de escopo

- Envio real de e-mail ou SMS.
- Dashboard administrativo.
- Templates persistidos.
- Filas externas.
- Webhooks de providers externos.
- Integracao com dominio de pedidos, pagamentos ou clientes.