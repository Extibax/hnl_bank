-- HNL Bank - Esquema de base de datos (PostgreSQL)
-- Este archivo documenta el esquema que el backend aplica en su migracion
-- embebida (backend/internal/repository/postgres.go). Se provee como entregable.

-- Usuarios / autenticacion
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Cuentas bancarias (metadatos en PG + referencia al libro mayor TigerBeetle)
CREATE TABLE IF NOT EXISTS user_accounts (
    id             UUID PRIMARY KEY,
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_number TEXT UNIQUE NOT NULL,
    tigerbeetle_id BYTEA NOT NULL,
    account_type   TEXT NOT NULL,
    currency       TEXT NOT NULL DEFAULT 'USD',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Historial de transacciones (metadata en PG para consulta/paginacion)
CREATE TABLE IF NOT EXISTS transactions (
    id            UUID PRIMARY KEY,
    from_user_id  UUID,
    to_user_id    UUID,
    from_account  TEXT,
    to_account    TEXT,
    amount        BIGINT NOT NULL,
    type          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT now(),
    status        TEXT NOT NULL DEFAULT 'completed'
);

CREATE INDEX IF NOT EXISTS idx_transactions_from_user ON transactions(from_user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_to_user   ON transactions(to_user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_from_acct ON transactions(from_account);
CREATE INDEX IF NOT EXISTS idx_transactions_to_acct   ON transactions(to_account);
