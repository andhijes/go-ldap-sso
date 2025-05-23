-- Tabel employees
CREATE TABLE employees (
    id SERIAL PRIMARY KEY,
    uid VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(150) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

-- Tabel scopes
CREATE TABLE scopes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

-- Tabel employee_scopes (many-to-many)
CREATE TABLE employee_scopes (
    employee_id INT REFERENCES employees(id) ON DELETE CASCADE,
    scope_id INT REFERENCES scopes(id) ON DELETE CASCADE,
    PRIMARY KEY (employee_id, scope_id)
);

-- Tabel token_blacklist (optional untuk revoke)
CREATE TABLE token_blacklist (
    token TEXT PRIMARY KEY,
    expires_at TIMESTAMP NOT NULL
);