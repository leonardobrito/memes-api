CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    auth_token VARCHAR(64) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS token_transactions (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    amount INTEGER NOT NULL,
    transaction_type VARCHAR(10) NOT NULL CHECK (transaction_type IN ('PURCHASE', 'USAGE')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_token_transactions_client_id ON token_transactions(client_id);
CREATE INDEX idx_clients_auth_token ON clients(auth_token);
