CREATE TABLE IF NOT EXISTS users
(
    id       SERIAL PRIMARY KEY,
    login    TEXT NOT NULL UNIQUE,
    password TEXT NOT NULL
);

CREATE TYPE status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');

CREATE TABLE IF NOT EXISTS orders
(
    id          SERIAL PRIMARY KEY,
    order_id    BIGINT    NOT NULL UNIQUE,
    user_id     INTEGER   NOT NULL,
    status      status    NOT NULL DEFAULT 'NEW',
    accrual     numeric(8, 2)      DEFAULT 0,
    uploaded_at TIMESTAMP NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE IF NOT EXISTS withdrawals
(
    id           SERIAL PRIMARY KEY,
    user_id      INTEGER   NOT NULL,
    order_id     BIGINT    NOT NULL UNIQUE,
    sum          numeric(8, 2)      DEFAULT 0,
    processed_at TIMESTAMP NOT NULL DEFAULT now(),
    FOREIGN KEY (user_id) REFERENCES users (id)
);
