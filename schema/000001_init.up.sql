-- Таблица пользователей
CREATE TABLE users
(
    id        SERIAL PRIMARY KEY,
    firstname TEXT          NOT NULL,
    username  TEXT          NOT NULL,
    user_id   BIGINT UNIQUE NOT NULL -- user_id из Telegram
);

-- Таблица кошельков
CREATE TABLE wallets
(
    id          SERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    private_key TEXT   NOT NULL,
    public_key  TEXT   NOT NULL
);