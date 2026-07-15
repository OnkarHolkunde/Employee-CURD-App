CREATE TABLE IF NOT EXISTS employees (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,

    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    company_name VARCHAR(150),
    address VARCHAR(255),
    city VARCHAR(100),
    county VARCHAR(100),
    postal VARCHAR(20),
    phone VARCHAR(30),
    email VARCHAR(150),
    web VARCHAR(255),

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    INDEX idx_employees_email (email)
);