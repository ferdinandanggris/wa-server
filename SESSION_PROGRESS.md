# WhatsApp Gateway — Session Progress

## Project
Multi-Tenant WhatsApp Message Gateway (Go 1.21+) — microservices architecture with RabbitMQ, PostgreSQL, and real WhatsApp API.

## Current Phase
**Phase 1: MVP** — Inbound/Outbound/Worker Pool ✅

---

## Completed Milestones

| # | Milestone | Status | Date |
|---|-----------|--------|------|
| 1 | Database schema (8 migrations) | ✅ | Week 1 |
| 2 | WhatsApp API client + phone normalization | ✅ | Week 1 |
| 3 | Inbound webhook (verify + receive messages) | ✅ | Week 1 |
| 4 | Outbound API (direct send + queue publish) | ✅ | Week 1 |
| 5 | Worker pool (RabbitMQ consumer → WhatsApp) | ✅ | Week 1 |
| 6 | RabbitMQ routing key fix (was using queueName as routingKey) | ✅ | May 18 |
| 7 | WebSocket hub for real-time updates | ✅ | Week 1 |
| 8 | DB status persistence (timestamps, error messages) | ✅ | May 18 |
| 9 | All golangci-lint issues fixed (zero warnings) | ✅ | May 18 |

## Architecture

```
WhatsApp User
    ↕ (webhook)
POST /webhook → WhatsAppHandler → MessageRepo (save)
                                  → Publisher (queue)
                                  → WebSocketHub (broadcast)
    ↕ (API)
POST /api/v1/messages → OutboundHandler → MessageRepo (save)
                                        → WhatsAppClient (direct)
                                        → Publisher (queue)
                                           ↓
                                     WorkerPool (consume)
                                        → WhatsAppClient (send)
                                        → MessageRepo (update status)
    ↕ (status webhook)
POST /webhook → processStatus → MessageRepo (update delivery timestamps)
```

## Key Decisions

- **Direct send + queue publish**: Handler sends immediately AND publishes to queue (double-send by design for testing; remove direct send in production)
- **Phone normalization**: 08xx → 628xx prefix
- **Default company ID**: `00000000-0000-0000-0000-000000000001`
- **Test conversation**: `55555555-5555-5555-5555-555555555555`
- **Webhook verify_token**: configured in WhatsApp dashboard

## Known Issues

| Issue | Severity | Notes |
|-------|----------|-------|
| Double-send (direct + queue) | Should-fix | Remove direct send path for production |
| `os.Exit` skips deferred cleanup | Should-fix | Replace with graceful shutdown |
| Missing doc comments on exports | Should-fix | All exported types/functions need godoc |
| WebSocket double-close panic | Should-fix | `client.Send` channel race condition |
| Dynamic query builder fragility | Nit | `fmt.Sprint` param numbering in `List()` |

## Environment

| Service | Connection |
|---------|-----------|
| Server | `localhost:9090` |
| PostgreSQL | `localhost:5432` — user: `wachat`, db: `wa_gateway` |
| RabbitMQ | `localhost:5672` — user: `wachat` |
| Ngrok | `https://fb6f-125-164-233-231.ngrok-free.app/webhook` |

## Next Steps (Phase 2)

- [ ] Template messages (WhatsApp template API)
- [ ] Billing / quota tracking
- [ ] JWT authentication
- [ ] API Keys for M2M
- [ ] Remove direct send (rely on worker pool only)
- [ ] Add doc comments on all exports
- [ ] WebSocket dedup + proper close handling
