CREATE TABLE domains (
    id TEXT PRIMARY KEY,
    account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE RESTRICT,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    php_version TEXT NOT NULL DEFAULT '8.3',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX domains_account_id_idx ON domains(account_id);
CREATE INDEX domains_node_id_idx ON domains(node_id);

CREATE TABLE domain_aliases (
    id TEXT PRIMARY KEY,
    domain_id TEXT NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE,
    status TEXT NOT NULL CHECK (status IN ('pending', 'active', 'disabled', 'error')),
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE INDEX domain_aliases_domain_id_idx ON domain_aliases(domain_id);
