CREATE TABLE
IF NOT EXISTS users
(
    id VARCHAR
(50) PRIMARY KEY,
    name VARCHAR
(255) NOT NULL,
    email VARCHAR
(255) UNIQUE NOT NULL,
    password_hash VARCHAR
(255) NOT NULL,
    phone_number VARCHAR
(20) NOT NULL,
    address_line1 VARCHAR
(255) NOT NULL,
    address_line2 VARCHAR
(255),
    address_line3 VARCHAR
(255),
    address_town VARCHAR
(100) NOT NULL,
    address_county VARCHAR
(100) NOT NULL,
    address_postcode VARCHAR
(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW
(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW
(),
    deleted_at TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users(deleted_at);
