## Goal
Build a Multi-Tenant WhatsApp Message Gateway — all phases complete.

## Constraints & Preferences
- Standard Go project structure (cmd/, internal/, migrations/)
- Docker deployment with PostgreSQL 15+, RabbitMQ, Redis
- JWT for user interaction, API Keys for M2M
- Template management uses **WABA ID** (not Phone Number ID) for Meta Graph API
- Pricing cost drawn from Meta's `pricing_analytics` endpoint per WABA
- Quota enforcement: hard block before send (atomic `UPDATE … WHERE condition`)
- Periodic background sync for phone numbers + pricing + billing reconciliation
- Phone numbers imported **only from Meta API** (`GET /{version}/{waba_id}/phone_numbers`)
- Admin assigns phone number to company via dashboard (manual)
- Outgoing message proceeds even if `company_id` is null (skip quota check)
- All companies share one WABA ID globally (in config)

## Progress
### Done
- **Phase 1 (Messaging + Worker Pool)** — Inbound/Outbound messaging, Worker Pool, WebSocket, graceful shutdown
- **Phase 2 (Billing)** — Quota enforcement, cost sync from Meta, billing logs
- **Phase 3 (Multi-Phone Number)** — phone_numbers table, sync from Meta, worker uses phone_number_id
- **Phase 4 (Pricing Analytics)** — waba_pricing table, pricing_analytics endpoint, monthly summary by category
- **Idempotency** — Inbound dedup via message_id, outbound via idempotency_key, migration 014
- **Infrastructure** — WABA Token Bucket rate limiter, Prometheus /metrics endpoint, Redis agent heartbeats
- **Code quality** — All tests pass, gofmt, go vet, go build clean

### In Progress
- *(none)*

### Blocked
- *(none)*

## Key Decisions
- **WABA ID for templates** — Message sending uses Phone Number ID; template management uses WABA ID
- **`pricing_analytics` for actual cost** — Stored in separate `waba_pricing` table (not billing_logs)
- **Idempotency** — Inbound: Meta's `message_id`; Outbound: client-provided `idempotency_key`
- **Atomic quota check+increment** — `UPDATE companies SET quota_used = quota_used + $2 WHERE id = $1 AND quota_used + $2 <= quota_limit`
- **Periodic sync combined** — Phone numbers + pricing + billing in one goroutine
- **Worker uses phone_number from DB** — Lookup by phone number resolves to Meta's `phone_number_id`
- **null company_id flow** — If phone not mapped to company, outgoing succeeds (no quota check)

## Next Steps
- *(none — all phases + infrastructure complete)*
