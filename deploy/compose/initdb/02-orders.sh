#!/bin/bash
set -euo pipefail
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname orders <<'SQL'
REVOKE ALL ON DATABASE orders FROM PUBLIC;
GRANT CONNECT, TEMP ON DATABASE orders TO orders_owner;
CREATE TABLE IF NOT EXISTS orders (
    id integer PRIMARY KEY,
    status text NOT NULL,
    customer text NOT NULL
);
ALTER TABLE orders OWNER TO orders_owner;
INSERT INTO orders (id, status, customer)
VALUES (1042, 'shipped', 'Ada Lovelace')
ON CONFLICT (id) DO NOTHING;
SQL
