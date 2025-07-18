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



INSERT INTO users (telegram_id, username,first_name,last_name)
VALUES (000000001,'Owner','Islam','Dashuev');

INSERT INTO wallets (user_id,private_key,address)
VALUES (000000001,'3b15c416ee5c4515e3fd72a382caa7c9bee2cfae5ad8416dacc8883712be08be','51esKm1ZXu8Tp51F53HM4ZBVgA1xR');

INSERT INTO balances (wallet_id,token_symbol,amount)
VALUES (1,'USDT',1000.0);

-- Добавим тестового пользователя
INSERT INTO users (telegram_id, username, first_name, last_name, created_at)
VALUES (123456789, 'testuser', 'Test', 'User', NOW());

-- Добавим тестовый кошелек
INSERT INTO wallets (user_id, private_key, address, created_at)
VALUES (123456789, 'testprivkey', 'TTESTADDRESS123', NOW());

-- Добавим баланс USDT для кошелька
INSERT INTO balances (wallet_id, token_symbol, amount, updated_at)
VALUES ((SELECT id FROM wallets WHERE address = 'TTESTADDRESS123'), 'USDT', 100.0, NOW());

-- Добавим виртуальные списания
INSERT INTO usdt_virtual_transfers (wallet_id, amount, status, created_at, processed_at)
VALUES
  ((SELECT id FROM wallets WHERE address = 'TTESTADDRESS123'), 5.0, 'pending', NOW() - INTERVAL '2 days', NULL),
  ((SELECT id FROM wallets WHERE address = 'TTESTADDRESS123'), 10.0, 'processed', NOW() - INTERVAL '5 days', NOW() - INTERVAL '4 days');

-- Добавим реальные списания (транзакции)
INSERT INTO transactions (from_wallet_id, to_address, token_symbol, amount, tx_hash, status, created_at)
VALUES
  ((SELECT id FROM wallets WHERE address = 'TTESTADDRESS123'), 'TDEST1', 'USDT', 20.0, '0xHASH1', 'success', NOW() - INTERVAL '10 days'),
  ((SELECT id FROM wallets WHERE address = 'TTESTADDRESS123'), 'TDEST2', 'USDT', 15.0, '0xHASH2', 'success', NOW() - INTERVAL '3 days');