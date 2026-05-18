-- Seed data for WhatsApp Gateway
-- Idempotent: uses INSERT ON CONFLICT DO NOTHING

-- Companies
INSERT INTO companies (id, name, code, phone_number, address, is_active, quota_limit, quota_used, created_at, updated_at)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'Default Company', 'default', '6282131955087', 'Jakarta, Indonesia', true, 50000, 0, NOW(), NOW()),
    ('11111111-1111-1111-1111-111111111111', 'Test Company', 'TEST', '6281234567890', 'Surabaya, Indonesia', true, 1000, 0, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Users
INSERT INTO users (id, company_id, email, password_hash, name, role, is_active, created_at, updated_at)
VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', '00000000-0000-0000-0000-000000000001', 'admin@default.com', '$2a$10$placeholder_hash_for_testing', 'Admin Default', 'admin', true, NOW(), NOW()),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '11111111-1111-1111-1111-111111111111', 'admin@test.com', '$2a$10$placeholder_hash_for_testing', 'Admin Test', 'admin', true, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Agents
INSERT INTO agents (id, company_id, user_id, name, email, status, max_concurrent_chats, created_at, updated_at)
VALUES
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', '00000000-0000-0000-0000-000000000001', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Agent A', 'agent.a@default.com', 'online', 10, NOW(), NOW()),
    ('dddddddd-dddd-dddd-dddd-dddddddddddd', '11111111-1111-1111-1111-111111111111', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Agent B', 'agent.b@test.com', 'offline', 5, NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Contacts
INSERT INTO contacts (id, company_id, wa_id, phone_number, name, profile_picture_url, is_blocked, created_at, updated_at)
VALUES
    ('33333333-3333-3333-3333-333333333333', '00000000-0000-0000-0000-000000000001', '6282131955087', '6282131955087', 'Ferdinand', NULL, false, NOW(), NOW()),
    ('44444444-4444-4444-4444-444444444444', '00000000-0000-0000-0000-000000000001', '628123456789', '628123456789', 'Budi', NULL, false, NOW(), NOW()),
    ('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', 'wa6281234567890', '6281234567890', 'Test Contact', NULL, false, NOW(), NOW())
ON CONFLICT (company_id, wa_id) DO NOTHING;

-- Conversations
INSERT INTO conversations (id, company_id, contact_id, assigned_agent_id, status, is_24h_window_active, unread_count, started_at, created_at, updated_at)
VALUES
    ('55555555-5555-5555-5555-555555555555', '00000000-0000-0000-0000-000000000001', '33333333-3333-3333-3333-333333333333', NULL, 'open', true, 1, NOW() - INTERVAL '1 day', NOW() - INTERVAL '1 day', NOW()),
    ('66666666-6666-6666-6666-666666666666', '00000000-0000-0000-0000-000000000001', '44444444-4444-4444-4444-444444444444', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 'assigned', true, 0, NOW() - INTERVAL '2 days', NOW() - INTERVAL '2 days', NOW()),
    ('77777777-7777-7777-7777-777777777777', '11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', NULL, 'open', false, 3, NOW() - INTERVAL '3 days', NOW() - INTERVAL '3 days', NOW())
ON CONFLICT (id) DO NOTHING;

-- Templates
INSERT INTO templates (id, wa_template_id, name, language, category, content, header_type, header_content, is_verified, meta_status, created_at, updated_at)
VALUES
    ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', 'welcome_template_id', 'welcome_greeting', 'en_US', 'MARKETING', 'Hello {{1}}, welcome to our service!', 'TEXT', 'Welcome', true, 'APPROVED', NOW(), NOW()),
    ('ffffffff-ffff-ffff-ffff-ffffffffffff', 'order_template_id', 'order_confirmation', 'id', 'UTILITY', 'Pesanan {{1}} Anda telah dikonfirmasi. Terima kasih!', 'TEXT', 'Konfirmasi Pesanan', true, 'APPROVED', NOW(), NOW())
ON CONFLICT (id) DO NOTHING;

-- Messages
INSERT INTO messages (id, conversation_id, message_id, direction, message_type, content, status, created_at)
VALUES
    ('aaaaaaaa-1111-1111-1111-111111111111', '55555555-5555-5555-5555-555555555555', 'msg_inbound_1', 'inbound', 'text', 'Halo, ada yang bisa dibantu?', 'delivered', NOW() - INTERVAL '23 hours'),
    ('aaaaaaaa-2222-2222-2222-222222222222', '55555555-5555-5555-5555-555555555555', 'msg_outbound_1', 'outbound', 'text', 'Halo! Ada yang bisa kami bantu hari ini?', 'delivered', NOW() - INTERVAL '22 hours'),
    ('aaaaaaaa-3333-3333-3333-333333333333', '66666666-6666-6666-6666-666666666666', 'msg_inbound_2', 'inbound', 'text', 'Saya ingin menanyakan status pesanan', 'delivered', NOW() - INTERVAL '2 days'),
    ('aaaaaaaa-4444-4444-4444-444444444444', '66666666-6666-6666-6666-666666666666', 'msg_outbound_2', 'outbound', 'text', 'Baik, akan kami cek status pesanan Anda', 'sent', NOW() - INTERVAL '2 days'),
    ('aaaaaaaa-5555-5555-5555-555555555555', '77777777-7777-7777-7777-777777777777', 'msg_inbound_3', 'inbound', 'text', 'Test message from Test Company', 'delivered', NOW() - INTERVAL '3 days'),
    ('aaaaaaaa-6666-6666-6666-666666666666', '77777777-7777-7777-7777-777777777777', 'msg_outbound_3', 'outbound', 'template', '', 'sent', NOW() - INTERVAL '3 days')
ON CONFLICT (message_id) DO NOTHING;

-- Billing Logs
INSERT INTO billing_logs (id, company_id, template_id, conversation_id, message_id, template_cost, created_at)
VALUES
    ('bbbbbbbb-1111-1111-1111-111111111111', '00000000-0000-0000-0000-000000000001', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '55555555-5555-5555-5555-555555555555', 'aaaaaaaa-1111-1111-1111-111111111111', 0.0500, NOW()),
    ('bbbbbbbb-2222-2222-2222-222222222222', '00000000-0000-0000-0000-000000000001', 'ffffffff-ffff-ffff-ffff-ffffffffffff', '66666666-6666-6666-6666-666666666666', 'aaaaaaaa-4444-4444-4444-444444444444', 0.0350, NOW() - INTERVAL '1 day'),
    ('bbbbbbbb-3333-3333-3333-333333333333', '11111111-1111-1111-1111-111111111111', 'eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', '77777777-7777-7777-7777-777777777777', 'aaaaaaaa-6666-6666-6666-666666666666', 0.0500, NOW() - INTERVAL '2 days')
ON CONFLICT (id) DO NOTHING;
