CREATE TABLE urls (
    id UUID PRIMARY KEY,
    short_url VARCHAR(255) NOT NULL UNIQUE,
    original_url VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE INDEX idx_short_url on urls(short_url);