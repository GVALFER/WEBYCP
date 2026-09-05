CREATE TABLE dns_providers (
    id TEXT PRIMARY KEY,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    driver TEXT NOT NULL CHECK (driver IN ('powerdns')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (node_id, driver)
);

CREATE TABLE dns_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    primary_nameserver TEXT NOT NULL,
    secondary_nameserver TEXT NOT NULL,
    default_ttl INTEGER NOT NULL CHECK (default_ttl BETWEEN 60 AND 86400),
    updated_at INTEGER NOT NULL
);

INSERT INTO dns_settings (
    id,
    primary_nameserver,
    secondary_nameserver,
    default_ttl,
    updated_at
) VALUES (1, '', '', 3600, unixepoch() * 1000);

CREATE TABLE dns_zones (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    provider_id TEXT NOT NULL REFERENCES dns_providers(id) ON DELETE RESTRICT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    primary_nameserver TEXT NOT NULL,
    secondary_nameserver TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'deleting', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX dns_zones_account_id_idx ON dns_zones(account_id);
CREATE INDEX dns_zones_provider_id_idx ON dns_zones(provider_id);

CREATE TABLE dns_records (
    id TEXT PRIMARY KEY,
    zone_id TEXT NOT NULL REFERENCES dns_zones(id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE,
    type TEXT NOT NULL CHECK (type IN ('A', 'AAAA', 'CNAME', 'MX', 'TXT')),
    content TEXT NOT NULL,
    ttl INTEGER NOT NULL CHECK (ttl BETWEEN 60 AND 86400),
    priority INTEGER NOT NULL CHECK (priority BETWEEN 0 AND 65535),
    synced_name TEXT NOT NULL,
    synced_type TEXT NOT NULL CHECK (synced_type IN ('', 'A', 'AAAA', 'CNAME', 'MX', 'TXT')),
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'deleting', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE (zone_id, name, type, content, priority)
);

CREATE INDEX dns_records_zone_id_idx ON dns_records(zone_id);
CREATE INDEX dns_records_rrset_idx ON dns_records(zone_id, name, type);
