-- Таблица виртуальных списаний USDT
CREATE TABLE usdt_virtual_transfers (
    id SERIAL PRIMARY KEY,
    wallet_id INTEGER NOT NULL REFERENCES wallets (id) ON DELETE CASCADE,
    amount NUMERIC(30, 6) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 'pending' или 'processed'
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMP
);

-- Индекс для быстрого поиска по wallet_id и статусу
CREATE INDEX idx_virtual_transfers_wallet_status ON usdt_virtual_transfers(wallet_id, status);

-- -- Тестовые данные для реального адреса
-- INSERT INTO users (telegram_id, username, first_name, last_name)
-- VALUES (888888888, 'realuser', 'Real', 'User')
-- ON CONFLICT (telegram_id) DO NOTHING;

-- INSERT INTO wallets (user_id, private_key, address)
-- VALUES (888888888, 'real_private_key', 'TG2FN9BxfTjX41tAyTeRTnqqrDKtpjyfEn')
-- ON CONFLICT (address) DO NOTHING;
