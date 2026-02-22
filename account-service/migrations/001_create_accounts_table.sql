CREATE TABLE
IF NOT EXISTS accounts
(
    account_number VARCHAR
(8) PRIMARY KEY,
    user_id VARCHAR
(50) NOT NULL,
    sort_code VARCHAR
(10) DEFAULT '10-10-10',
    name VARCHAR
(255) NOT NULL,
    account_type VARCHAR
(20) DEFAULT 'personal',
    balance DECIMAL
(10,2) DEFAULT 0.00,
    currency VARCHAR
(3) DEFAULT 'GBP',
    created_at TIMESTAMP NOT NULL DEFAULT NOW
(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW
(),
    deleted_at TIMESTAMP,
    CONSTRAINT balance_positive CHECK
(balance >= 0)
);

CREATE INDEX idx_accounts_user_id ON accounts(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_accounts_deleted_at ON accounts(deleted_at);
