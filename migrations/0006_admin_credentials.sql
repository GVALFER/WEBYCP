ALTER TABLE users ADD COLUMN username TEXT NOT NULL COLLATE NOCASE DEFAULT '';
ALTER TABLE users ADD COLUMN must_change_password INTEGER NOT NULL DEFAULT 0
    CHECK (must_change_password IN (0, 1));

UPDATE users
SET username = CASE
    WHEN id = (
        SELECT id
        FROM users
        WHERE role = 'admin'
        ORDER BY created_at, id
        LIMIT 1
    ) THEN 'admin'
    ELSE 'user_' || substr(lower(id), 1, 27)
END;

CREATE UNIQUE INDEX users_username_idx ON users(username);
