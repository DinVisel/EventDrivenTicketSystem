CREATE TABLE tickets (
    id SERIAL PRIMARY KEY,
    event_name VARCHAR(255) NOT NULL,
    stock INT NOT NULL
);

INSERT INTO tickets (event_name, stock) VALUES ('Konser 2026', 100);