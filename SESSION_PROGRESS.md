# WhatsApp Gateway — Session Progress

## Project
Multi-Tenant WhatsApp Message Gateway (Go 1.21+) — microservices architecture with RabbitMQ, PostgreSQL, and Meta WhatsApp Cloud API.

## Current Phase
**All Phases Complete** — Full backend ready for WinForms Admin App.

---

## Completed Milestones

| # | Milestone | Status | Date |
|---|-----------|--------|------|
| 1 | Database schema (8+ migrations) | ✅ | Week 1 |
| 2 | WhatsApp API client + phone normalization | ✅ | Week 1 |
| 3 | Inbound webhook (verify + receive messages) | ✅ | Week 1 |
| 4 | Outbound API (direct send + queue publish) | ✅ | Week 1 |
| 5 | Worker pool (RabbitMQ consumer → WhatsApp) | ✅ | Week 1 |
| 6 | RabbitMQ routing key fix | ✅ | May 18 |
| 7 | WebSocket hub for real-time updates | ✅ | Week 1 |
| 8 | DB status persistence (timestamps, error messages) | ✅ | May 18 |
| 9 | All golangci-lint issues fixed (zero warnings) | ✅ | May 18 |
| 10 | Idempotency (inbound dedup + outbound idempotency_key) | ✅ | May 18 |
| 11 | Template management (CRUD via WhatsApp API) | ✅ | May 18 |
| 12 | Billing system (quota enforcement, cost sync from Meta) | ✅ | May 18 |
| 13 | Multi-Phone Number support (sync from Meta, DB + API) | ✅ | May 18 |
| 14 | Pricing Analytics (pricing_analytics from Meta) | ✅ | May 18 |
| 15 | WABA Token Bucket rate limiter | ✅ | May 18 |
| 16 | Prometheus /metrics endpoint | ✅ | May 18 |
| 17 | Redis agent heartbeats | ✅ | May 18 |
| 18 | JWT authentication + role-based access (superadmin/admin/cs) | ✅ | May 18 |
| 19 | Company management (CRUD API) | ✅ | May 18 |
| 20 | User management (CRUD + Login API) | ✅ | May 18 |
| 21 | Seed superadmin user (superadmin@wa.com / superadmin123) | ✅ | May 19 |
| 22 | Phone number assign to company (POST /{id}/assign) | ✅ | May 19 |
| 23 | Phone number profile (GET/PUT /{id}/profile via Meta API) | ✅ | May 19 |

---

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

Admin API:
    POST /api/v1/auth/login         → JWT token
    GET|POST /api/v1/companies      → CRUD (superadmin only create)
    GET|POST /api/v1/users           → CRUD
    GET|POST /api/v1/phone-numbers  → List/Sync
    POST /{id}/assign               → Assign phone to company
    GET|PUT /{id}/profile           → WhatsApp Business Profile
    GET /api/v1/billing/*           → Usage, Quota, Cost summary
    GET /api/v1/pricing/*           → Pricing analytics
```

## Key Decisions

| Decision | Current State |
|----------|---------------|
| **Auth** | JWT for user interaction, role-based (superadmin/admin/cs) |
| **Phone number assignment** | Admin assigns phone to company via dashboard (manual) |
| **Pricing analytics** | Uses Meta's `pricing_analytics` endpoint per WABA |
| **Idempotency** | Inbound via Meta message_id, outbound via idempotency_key |
| **WABA ID** | All companies share one WABA ID (configured in .env) |
| **Quota enforcement** | Atomic UPDATE with WHERE condition (before send) |
| **Direct send** | Still active (double-send with queue) — remove for production |

## Environment

| Service | Connection |
|---------|-----------|
| Server | `localhost:9090` |
| PostgreSQL | `localhost:5432` — user: `wachat`, db: `wa_gateway` |
| RabbitMQ | `localhost:5672` — user: `wachat` |
| Redis | `localhost:6379` |

## New API Endpoints (May 19)

| Endpoint | Method | Description | Auth |
|----------|--------|-------------|------|
| `POST /api/v1/phone-numbers/{id}/assign` | POST | Assign phone to company | Superadmin |
| `GET /api/v1/phone-numbers/{id}/profile` | GET | Get WhatsApp Business Profile | Auth |
| `PUT /api/v1/phone-numbers/{id}/profile` | PUT | Update profile (about, desc, etc) | Auth |

## Next Steps

- [ ] **WinForms Admin App (.NET Framework 4.8)**
  - [ ] Company management UI
  - [ ] User management UI
  - [ ] Phone number list + assign UI
  - [ ] Phone number profile update UI
  - [ ] Analytics / Dashboard UI
  - [ ] JWT login + token storage
- [ ] Production: Remove direct send path (rely on worker pool only)
