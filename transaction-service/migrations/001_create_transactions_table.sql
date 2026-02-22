CREATE TABLE
IF NOT EXISTS transactions
(
    id VARCHAR
(50) PRIMARY KEY,
    account_number VARCHAR
(8) NOT NULL,
    user_id VARCHAR
(50) NOT NULL,
    amount DECIMAL
(10,2) NOT NULL,
    currency VARCHAR
(3) DEFAULT 'GBP',
    type VARCHAR
(20) NOT NULL,
    reference VARCHAR
(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW
(),
    CONSTRAINT amount_positive CHECK
(amount > 0)
);

CREATE INDEX idx_transactions_account ON transactions(account_number);
CREATE INDEX idx_transactions_user ON transactions(user_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at);
