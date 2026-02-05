CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    applied_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS stats_total (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    words_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS stats_daily (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    username TEXT NOT NULL,
    words_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id, day_date)
);

CREATE INDEX IF NOT EXISTS idx_stats_daily_chat_day ON stats_daily(chat_id, day_date);
CREATE INDEX IF NOT EXISTS idx_stats_total_chat ON stats_total(chat_id);
