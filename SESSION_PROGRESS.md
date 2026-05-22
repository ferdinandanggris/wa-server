# WhatsApp Gateway — Session Progress

## Project
Multi-Tenant WhatsApp Message Gateway (Go 1.21+ for backend) + WinForms Admin App (.NET Framework 4.8) + React Vite Chat

## Current Phase
**Phase 7: React Vite Chat Integration** — ✅ UI Dashboard + WinForms integration ready

## Known Issues (Next Session)
| # | Issue | Notes |
|---|-------|-------|
| 1 | **Monitor tab** — Inbox/Outbox still empty placeholders |
| 2 | **Phone Numbers tab** — Missing `lblServiceTitle` label |
| 3 | **Cancel/Esc revert** — Not implemented |
| 4 | **React Vite Chat** — Not started |

---

## Completed Backend Milestones

| # | Milestone | Status | Date |
|---|-----------|--------|------|
| 1 | Database schema (8+ migrations) | ✅ | Week 1 |
| 2 | WhatsApp API client + phone normalization | ✅ | Week 1 |
| 3 | Inbound webhook (verify + receive messages) | ✅ | Week 1 |
| 4 | Outbound API (direct send + queue publish) | ✅ | Week 1 |
| 5 | Worker pool (RabbitMQ consumer → WhatsApp) | ✅ | Week 1 |
| 6 | RabbitMQ routing key fix | ✅ | May 18 |
| 7 | WebSocket hub for real-time updates | ✅ | Week 1 |
| 8 | Idempotency (inbound dedup + outbound idempotency_key) | ✅ | May 18 |
| 9 | Template management (CRUD via WhatsApp API) | ✅ | May 18 |
| 10 | Billing system (quota enforcement, cost sync from Meta) | ✅ | May 18 |
| 11 | Multi-Phone Number support (sync from Meta, DB + API) | ✅ | May 18 |
| 12 | Pricing Analytics (pricing_analytics from Meta) | ✅ | May 18 |
| 13 | JWT authentication + role-based access | ✅ | May 18 |
| 14 | Company management (CRUD API) | ✅ | May 18 |
| 15 | User management (CRUD + Login API) | ✅ | May 18 |
| 16 | Phone number assign to company (POST /{id}/assign) | ✅ | May 19 |
| 17 | Phone number profile (GET/PUT /{id}/profile via Meta API) | ✅ | May 19 |
| 18 | Seed superadmin user | ✅ | May 19 |

---

## WinForms Admin App Plan

### Project Details

| Item | Value |
|------|-------|
| **Location** | `/mnt/d/project/wa-client` |
| **Framework** | .NET Framework 4.8 |
| **UI** | WinForms |
| **Project Type** | SPA (Single Panel Application) + TreeView Navigation |

### Tech Stack

| Component | Choice |
|-----------|--------|
| Framework | .NET Framework 4.8 |
| UI | WinForms |
| HTTP Client | HttpClient (built-in) |
| JSON | Newtonsoft.Json |
| Auth | JWT Bearer token |

### UI Structure

```
MainForm (Shell)
├── panelSidebar (TreeView)     ← Fixed width, left side
│   └── Phone Number
│       ├── +6281234567890 ●
│       └── +6289876543210 ○
└── panelContent (MainPageView) ← Fill, right side
    ├── tabDashboard
    ├── tabCompany      (DGV inline edit)
    ├── tabUser         (DGV inline edit)
    ├── tabPhone Numbers (DGV view-only)
    ├── tabMonitor      (Inbox/Outbox tabs)
    ├── tabLog
    └── tabAnalytics
```

### TreeView Behavior

| Action | Result |
|--------|--------|
| Click Phone Number (parent) | Expand/collapse nodes |
| Click Phone Number (child) | Navigate to Phone Numbers tab |
| Double-click Phone Number (child) | Open PhoneDetailView overlay |
| Right-click | Context Menu Strip |

### Context Menu Items

| Node | Menu Items |
|------|-----------|
| Phone Number (parent) | Sync from Meta, Refresh |
| Phone Number (child) | View Detail, Edit, Assign to Company, Update Profile |

### Main Content Area Pattern

- **DGV (DataGridView)** - inline edit
- **Row colors**: Putih (normal), Kuning (dirty/modified), Merah (will delete)
- **Keyboard Del** - mark row for delete (merah)
- **F5 / Tombol Refresh** - reload data from server
- **Simpan** - execute create/update/delete
- **Cancel / Esc** - revert all changes

### Login Flow

```
Program.cs
├── Check: JWT token exists?
│   ├── Yes → MainForm
│   └── No → LoginForm
└── LoginForm
    └── POST /api/v1/auth/login → Store JWT → MainForm
```

### Phases

| Phase | Description | Status |
|-------|-------------|--------|
| 1 | Project Setup + MainForm Shell + TreeView | ✅ Done |
| 2 | Login Form (simple fields) | ✅ Done |
| 3 | Company View (DGV inline edit) | ✅ Done |
| 4 | User View (DGV inline edit) | ✅ Done |
| 5 | Phone Number View + TreeView sync | ✅ Done |
| 6 | Analytics/Dashboard View | ✅ Done |
| 7 | React Vite Chat Integration | ⏳ Pending |

### API Endpoints to Consume

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | Login |
| `/api/v1/companies` | GET, POST | Company CRUD |
| `/api/v1/companies/{id}` | GET, PUT, DELETE | Company CRUD |
| `/api/v1/users` | GET, POST | User CRUD |
| `/api/v1/users/{id}` | GET, PUT, DELETE | User CRUD |
| `/api/v1/phone-numbers` | GET, POST | List + Sync |
| `/api/v1/phone-numbers/{id}/assign` | POST | Assign to company |
| `/api/v1/phone-numbers/{id}/profile` | GET, PUT | Profile |
| `/api/v1/billing/quota` | GET | Quota usage |
| `/api/v1/billing/cost-summary` | GET | Cost summary |

---

## Environment

| Service | Connection |
|---------|-----------|
| Server (Backend) | `localhost:9090` |
| PostgreSQL | `localhost:5432` — user: `wachat`, db: `wa_gateway` |
| RabbitMQ | `localhost:5672` — user: `wachat` |
| Redis | `localhost:6379` |

---

## Next Steps

- [x] Create WinForms project at `/mnt/d/project/wa-client`
- [x] Setup MainForm with panelSidebar + panelContent
- [x] Add TreeView with nodes: Phone Number
- [x] Create LoginForm
- [x] Implement JWT auth flow
- [x] Build Company View (inline DGV)
- [x] Build User View (inline DGV)
- [x] Build Phone Number View + TreeView sync
- [x] Build Analytics/Dashboard View
- [x] Phone context menu: Edit (ProfileEditForm), Assign to Company, Update Profile
- [x] Billing tab (BillingView) — backend wrapper fixed, client needs debugging
- [x] Debug BillingView API calls — fixed ApiClient error messages, date format, validation. DB columns added (phone_number, conversation_category). Both endpoints verified working via curl.
- [x] Phone Numbers inline edit — `dgvService` now has checkbox for `is_active`, CellValueChanged/KeyDown handlers, Save button calling PUT /{id}
- [ ] Monitor tab (Inbox/Outbox)
- [ ] Cancel/Esc revert pattern
- [x] Add React Vite chat integration (wa-chat)
- [x] WinForms: CS role → full-screen WebView2 with chat UI
- [x] WinForms: Ctrl+Shift+L logout for CS users
- [ ] Build wa-chat from build-chat.sh before running WinForms in CS mode

## Session Summary — May 20 2026

### Done
- Updated `SESSION_PROGRESS.md` to match reality (Phase 1-6 done)
- **MainForm.cs dead code removed**: ProcessCmdKey DashboardView branch, ShowView else, SyncPhoneNumbers stub, NodeMouseDoubleClick level-0
- **Company/User CRUD feedback**: Save now shows per-item errors + summary count
- **Phone context menu**: `Edit` → ProfileEditForm (GET/PUT /profile via Meta API), `Assign to Company` → AssignCompanyForm, `Update Profile` → read-only fetch
- **Billing tab**: `BillingView` UserControl embedded in tabLog (renamed to "Billing"), quota check + cost summary with date filters
- **Backend billing.go**: All 4 endpoints now use standard `{ok, data}` wrapper (was raw JSON), tests updated and passing

## Session Summary — May 21 2026

### Done
- **ApiClient.cs** — Error messages now include raw server response body for debugging
- **BillingView.cs** — Added validation message for empty company selection, error feedback for company load, fixed date format from `"o"` to RFC3339 (`yyyy-MM-ddTHH:mm:ssZ`)
- **Billing endpoints verified via curl** — Quota returns `{ok: true, data: {quota_limit, quota_used, remaining}}`, Cost Summary returns `{ok: true, data: [...]}`
- **DB fix** — Added missing `phone_number` and `conversation_category` columns to `billing_logs` table
- **Docker** — Rebuilt server image with billing wrapper fix, restarted container

## Session Summary — May 22 2026

### Done
- **MessageBubble alignment fix** — `normalizeMsg` uppercases direction (`"inbound"` → `"INBOUND"`) so `isOutbound` check works
- **Duplicate conversation fix** — Migration `016_add_unique_contact_phone.sql` adds `UNIQUE(contact_id, phone_number)`, dedup 6 rows. Webhook now uses `GetByPhoneNumberAndContact` instead of `GetByContactID` (works even when `company_id` is null)
- **Contact name from Meta profile** — Added `WhatsAppContact` struct + `Contacts` field to `WhatsAppValue` (was silently dropped). Webhook now parses `contacts[].profile.name` and always upserts contact with the name (both CREATE and UPDATE)
- **Phone number normalization** — New `internal/phone` package with `Normalize()` converting to `+62` format. Applied in `ensureConversation`, worker send path, and webhook `processMessage`
- **Docker** — Rebuilt with `docker compose build --no-cache` (direct `docker build` didn't update compose-managed image)

### Phone Numbers Inline Edit
- **Backend**: Added `PUT /api/v1/phone-numbers/{id}` endpoint to update `is_active`
  - Repository: `UpdateIsActive(ctx, id, isActive bool)`
  - Service: `UpdateIsActive(ctx, id, isActive bool) (*PhoneNumber, error)`
  - Handler: `updateIsActive` with `{is_active: bool}` request body
  - Route: `PUT /api/v1/phone-numbers/{id}` (guarded by authMW)
- **Frontend** (`MainPageView.cs` + Designer):
  - Added `DataGridViewCheckBoxColumn` for `is_active` on `dgvService`
  - Added `dgvService_CellValueChanged` → marks `IsDirty = true`
  - Added `dgvService_KeyDown` → Delete key toggles `IsDeleted`
  - Added `btnServiceSave` → iterates dirty/deleted phones, calls PUT/DELETE API
  - Row colors: yellow=dirty, pink=deleted, gray=inactive, white=active

### wa-chat (React Vite Chat UI)
- **Project moved**: `wa-client/wa-chat/` — Vite + React 19 + TypeScript + Tailwind CSS v3
- **Replicated old UI exactly**: Same 3-panel WhatsApp-style layout from `WaMeta.UI`
  - **ChatSidebar** (70px): App icons, unread badges, refresh, user avatar
  - **ConversationSidebar** (350px): Search bar, filter buttons (All/Unread/Read), conversation list with avatar, preview, time, unread badge, app/channel tags
  - **ChatWindow**: Header with avatar + name, message bubbles (green outbound, white inbound), ChatInput with emoji/attach/template/send, typing indicator, reply/reaction/resend actions, "Jendela Melayani Berakhir" banner for template-required conversations, empty state with "Whatsapp Client" branding
- **Dummy data**: 5 conversations + 8 messages across 2 apps
- **Dependencies**: Shadcn UI (Radix primitives: avatar, scroll-area, separator, tooltip, slot), lucide-react icons, clsx + tailwind-merge, class-variance-authority
- **Build**: `npm run build` → `wa-chat/dist/` (verified, 27KB CSS + 325KB JS)
- **Script**: `build-chat.sh` in `wa-client/`

### WinForms CS Role Integration
- **MainForm.cs**: Constructor now checks `AuthService.Instance.CurrentUser?.Role`
  - If `"cs"`: hides sidebar (`splitContainer`) + menu bar, creates `WebView2` (Dock Fill), navigates to `wa-chat/dist/index.html` via `SetVirtualHostNameToFolderMapping` (solves CORS)
  - If admin: existing behavior unchanged
- **LoginForm.cs**: No more "Login successful!" message box — goes straight to MainForm
- **CS Logout**: `Ctrl+Shift+L` keyboard shortcut triggers logout (since menu is hidden)
- **WebView2**: NuGet package `Microsoft.Web.WebView2` v1.0.2903.40 added to csproj
- **Designer**: Added `webViewChat` field declaration

## Session Summary — May 21 2026 (Part 2)

### Conversation Display Fixes
- **Bug: `onUnusedReset` typo** (`wa-chat/src/hooks/useMessages.ts:39,118`) — `markRead` was never called because the function name `onUnusedReset` didn't match the parameter `onUnreadReset`. Fixed both the call site and the fallback stub.
- **Empty field fallbacks** (`ConversationSidebar.tsx`, `ChatWindow.tsx`):
  - `customer_name` now falls back to `customer_wa_id` instead of "Unknown"
  - `getInitials` falls back to `customer_wa_id` for avatar when name is empty
- **Seed data** (`migrations/seed.sql`): Added `phone_number` and `last_message_preview` columns to conversations INSERT.
- **DB data backfill**: Ran UPDATE to populate empty `phone_number` and `last_message_preview` in existing conversations.

### Verified Name on Conversations
- **Backend repository** (`internal/repository/conversation.go`):
  - Added `VerifiedName` field to `ConversationRow` struct
  - Extended `conversationSelectWithJoin` query to `LEFT JOIN phone_numbers pn ON c.phone_number = pn.phone_number` and select `COALESCE(pn.verified_name, '')`
  - Updated `scanConversationRow` to scan the new column
- **Backend handler** (`internal/api/handlers/conversation.go`):
  - Added `VerifiedName` to `conversationResponse` struct with `json:"verified_name"`
  - Set `VerifiedName` in `convToResponse` from `row.VerifiedName`
- **TypeScript** (`wa-chat/src/types/chat.ts`): Added `verified_name: string` to `Conversation` interface
- **Frontend** (`ConversationSidebar.tsx`): Shows green `verified_name` badge next to "To: {display_number}" badge when non-empty
- **Docker**: Rebuilt `wa-server` Docker image with changes, restarted container

### Side Work
- **CORS headers** — Added `Access-Control-Max-Age: 0` and `Content-Length` to both `writeJSON` functions (main.go + helpers.go) to prevent chunked encoding issues
- **WebView2 CORS** — Added `--disable-web-security --disable-features=BlockInsecurePrivateNetworkRequests` to `CoreWebView2EnvironmentOptions` in `MainForm.cs`
- **Duplicate route fix** — Removed duplicate `/api/v1/conversations/` handler from `outbound.go` (both outbound.go and conversation.go registered it, causing startup panic)
- **SQL scan fixes** — Wrapped UUID `COALESCE` with `::text` cast, added COALESCE for nullable string columns
- **Phone number profile migration** — `015_add_phone_number_profile.sql`: 9 new columns for Business Profile data
- **Phone number model + repo** — `PhoneNumber` struct extended, `UpdateProfile` method, `Upsert` includes `verified_name`
- **Phone number service** — `SyncFromMeta` now fetches and stores Business Profile per number
- **Phone summary endpoint** — Returns `verified_name`, `is_active`, `about`, `profile_picture_url`
- **ChatSidebar** — Active dot (green/gray) indicator with tooltip showing verified name
