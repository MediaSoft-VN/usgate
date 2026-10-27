-- Run this query in the PostgreSQL database used by new-api when token creation
-- fails with a duplicate-key error on tokens_pkey. It advances the tokens.id
-- sequence to the current maximum ID without changing token rows.
SELECT setval(
    pg_get_serial_sequence('tokens', 'id'),
    COALESCE(MAX(id), 1),
    MAX(id) IS NOT NULL
)
FROM tokens;
