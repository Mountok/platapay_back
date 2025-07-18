-- Таблица пользователей
CREATE TABLE users
(
    id            SERIAL PRIMARY KEY,
    telegram_id   BIGINT UNIQUE NOT NULL, -- ID из Telegram
    username      TEXT,                   -- @username, может быть NULL
    first_name    TEXT,
    last_name     TEXT,
    created_at    TIMESTAMP DEFAULT NOW()
);

-- Таблица кошельков
CREATE TABLE wallets
(
    id          SERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    private_key TEXT    NOT NULL,
    address     TEXT    NOT NULL UNIQUE,
    created_at  TIMESTAMP DEFAULT NOW()
);

-- Таблица балансов
CREATE TABLE balances
(
    id           SERIAL PRIMARY KEY,
    wallet_id    INTEGER NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    token_symbol TEXT    NOT NULL,
    amount       NUMERIC(30, 6) DEFAULT 0,
    updated_at   TIMESTAMP      DEFAULT NOW(),
    UNIQUE (wallet_id, token_symbol)
);




-- Таблица транзакций
CREATE TABLE transactions
(
    id             SERIAL PRIMARY KEY,
    from_wallet_id INTEGER REFERENCES wallets (id),
    to_address   TEXT ,
    token_symbol   TEXT           NOT NULL,
    amount         NUMERIC(30, 6) NOT NULL,
    tx_hash        TEXT,
    status         TEXT      DEFAULT 'confirmed',
    created_at     TIMESTAMP DEFAULT NOW()
);

CREATE TABLE orderqr
(
    id SERIAL PRIMARY KEY,
    telegram_id BIGINT NOT NULL,
    qrcode TEXT NOT NULL,
    summa BIGINT NOT NULL,
    crypto NUMERIC(30, 6) NOT NULL,
    ispaid BOOLEAN NOT NULL DEFAULT false
);
