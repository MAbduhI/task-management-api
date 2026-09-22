-- Seed Users (Password: password123)
INSERT INTO users (id, uuid, name, email, password_hash)
VALUES
    (1, '11111111-1111-1111-1111-111111111111', 'Alice Admin', 'alice@example.com', '$2a$10$oSSKjIHAvSIaVab8DnoYeuoAFmacly1g1OIhzty/15WxxH7X5z6Ha'),
    (2, '22222222-2222-2222-2222-222222222222', 'Bob Developer', 'bob@example.com', '$2a$10$oSSKjIHAvSIaVab8DnoYeuoAFmacly1g1OIhzty/15WxxH7X5z6Ha'),
    (3, '33333333-3333-3333-3333-333333333333', 'Charlie Backend', 'charlie@example.com', '$2a$10$oSSKjIHAvSIaVab8DnoYeuoAFmacly1g1OIhzty/15WxxH7X5z6Ha'),
    (4, '44444444-4444-4444-4444-444444444444', 'David Product', 'david@example.com', '$2a$10$oSSKjIHAvSIaVab8DnoYeuoAFmacly1g1OIhzty/15WxxH7X5z6Ha')
ON CONFLICT (email) DO NOTHING;

-- Seed Teams
INSERT INTO teams (id, uuid, name)
VALUES
    (1, 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Backend Engineering Team'),
    (2, 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Product Management Team')
ON CONFLICT (uuid) DO NOTHING;

-- Seed Team Members
-- Alice, Bob, Charlie in Team 1; David in Team 2
INSERT INTO team_members (id, uuid, team_id, user_id, role)
VALUES
    (1, 'c1111111-1111-1111-1111-111111111111', 1, 1, 'ADMIN'),
    (2, 'c2222222-2222-2222-2222-222222222222', 1, 2, 'MEMBER'),
    (3, 'c3333333-3333-3333-3333-333333333333', 1, 3, 'MEMBER'),
    (4, 'c4444444-4444-4444-4444-444444444444', 2, 4, 'ADMIN')
ON CONFLICT (team_id, user_id) DO NOTHING;

-- Seed Tasks
INSERT INTO tasks (id, uuid, title, description, status, created_by, assignee_id)
VALUES
    (1, '9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d', 'Setup CI/CD Pipeline with GitHub Actions', 'Automate test suite, race condition checks, and container image builds.', 'IN_PROGRESS', 1, 2),
    (2, '8c2edb5e-4c8e-5cbe-acee-3c1e8c4eda7e', 'Implement Distributed Idempotency Key Lock', 'Prevent duplicate task creations with Redis atomic SETNX lock and 24h TTL.', 'DONE', 1, 3),
    (3, '7d3fec6f-5d9f-6dcf-bdff-4d2f9d5feb8f', 'Database Transaction & Audit Log on Assign', 'Ensure atomic task reassignment and audit trail logging in single transaction.', 'TODO', 1, NULL)
ON CONFLICT (uuid) DO NOTHING;

-- Update sequences to avoid collision on next inserts
SELECT setval('users_id_seq', (SELECT COALESCE(MAX(id), 1) FROM users));
SELECT setval('teams_id_seq', (SELECT COALESCE(MAX(id), 1) FROM teams));
SELECT setval('team_members_id_seq', (SELECT COALESCE(MAX(id), 1) FROM team_members));
SELECT setval('tasks_id_seq', (SELECT COALESCE(MAX(id), 1) FROM tasks));
