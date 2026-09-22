-- Remove all seed tasks, team members, teams, and users
DELETE FROM task_logs WHERE performed_by IN (1, 2, 3, 4);

DELETE FROM tasks
WHERE uuid IN (
    '9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d',
    '8c2edb5e-4c8e-5cbe-acee-3c1e8c4eda7e',
    '7d3fec6f-5d9f-6dcf-bdff-4d2f9d5feb8f'
);

DELETE FROM team_members
WHERE uuid IN (
    'c1111111-1111-1111-1111-111111111111',
    'c2222222-2222-2222-2222-222222222222',
    'c3333333-3333-3333-3333-333333333333',
    'c4444444-4444-4444-4444-444444444444'
);

DELETE FROM teams
WHERE uuid IN (
    'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
    'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb'
);

DELETE FROM users
WHERE email IN (
    'alice@example.com',
    'bob@example.com',
    'charlie@example.com',
    'david@example.com'
);
