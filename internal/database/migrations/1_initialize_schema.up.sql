CREATE TABLE IF NOT EXISTS urls (
    short_url VARCHAR(20) PRIMARY KEY,
    origin VARCHAR(2000) NOT NULL ,
    created_at TIMESTAMP NOT NULL,
    last_access TIMESTAMP,
    is_expired boolean not null default false
);

CREATE TABLE IF NOT EXISTS short_urls_access (
    access_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS long_urls_access (
    access_at TIMESTAMP NOT NULL
);