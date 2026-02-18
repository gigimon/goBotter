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

CREATE TABLE IF NOT EXISTS forward_given_total (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    forward_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS forward_given_daily (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    username TEXT NOT NULL,
    forward_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id, day_date)
);

CREATE TABLE IF NOT EXISTS forward_target_total (
    chat_id INTEGER NOT NULL,
    target_key TEXT NOT NULL,
    target_label TEXT NOT NULL,
    forward_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, target_key)
);

CREATE TABLE IF NOT EXISTS forward_target_daily (
    chat_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    target_key TEXT NOT NULL,
    target_label TEXT NOT NULL,
    forward_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, day_date, target_key)
);

CREATE INDEX IF NOT EXISTS idx_forward_given_daily_chat_day ON forward_given_daily(chat_id, day_date);
CREATE INDEX IF NOT EXISTS idx_forward_given_total_chat ON forward_given_total(chat_id);
CREATE INDEX IF NOT EXISTS idx_forward_target_daily_chat_day ON forward_target_daily(chat_id, day_date);
CREATE INDEX IF NOT EXISTS idx_forward_target_total_chat ON forward_target_total(chat_id);

CREATE TABLE IF NOT EXISTS reaction_given_total (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    reactions_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS reaction_given_daily (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    username TEXT NOT NULL,
    reactions_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id, day_date)
);

CREATE TABLE IF NOT EXISTS reaction_popular_total (
    chat_id INTEGER NOT NULL,
    reaction_key TEXT NOT NULL,
    reaction_label TEXT NOT NULL,
    reactions_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, reaction_key)
);

CREATE TABLE IF NOT EXISTS reaction_popular_daily (
    chat_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    reaction_key TEXT NOT NULL,
    reaction_label TEXT NOT NULL,
    reactions_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, day_date, reaction_key)
);

CREATE INDEX IF NOT EXISTS idx_reaction_given_daily_chat_day ON reaction_given_daily(chat_id, day_date);
CREATE INDEX IF NOT EXISTS idx_reaction_given_total_chat ON reaction_given_total(chat_id);
CREATE INDEX IF NOT EXISTS idx_reaction_popular_daily_chat_day ON reaction_popular_daily(chat_id, day_date);
CREATE INDEX IF NOT EXISTS idx_reaction_popular_total_chat ON reaction_popular_total(chat_id);

CREATE TABLE IF NOT EXISTS reaction_message_state (
    chat_id INTEGER NOT NULL,
    message_id INTEGER NOT NULL,
    reaction_key TEXT NOT NULL,
    reaction_label TEXT NOT NULL,
    last_total_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, message_id, reaction_key)
);
CREATE INDEX IF NOT EXISTS idx_reaction_message_state_chat_msg ON reaction_message_state(chat_id, message_id);

CREATE TABLE IF NOT EXISTS reaction_received_total (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    username TEXT NOT NULL,
    reactions_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS reaction_received_daily (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    username TEXT NOT NULL,
    reactions_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id, day_date)
);

CREATE INDEX IF NOT EXISTS idx_reaction_received_daily_chat_day ON reaction_received_daily(chat_id, day_date);
CREATE INDEX IF NOT EXISTS idx_reaction_received_total_chat ON reaction_received_total(chat_id);

CREATE TABLE IF NOT EXISTS reaction_received_type_total (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    reaction_key TEXT NOT NULL,
    reaction_label TEXT NOT NULL,
    reactions_total INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id, reaction_key)
);

CREATE TABLE IF NOT EXISTS reaction_received_type_daily (
    chat_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    day_date TEXT NOT NULL,
    reaction_key TEXT NOT NULL,
    reaction_label TEXT NOT NULL,
    reactions_count INTEGER NOT NULL DEFAULT 0,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, user_id, day_date, reaction_key)
);

CREATE INDEX IF NOT EXISTS idx_reaction_received_type_total_chat_user ON reaction_received_type_total(chat_id, user_id);
CREATE INDEX IF NOT EXISTS idx_reaction_received_type_daily_chat_user_day ON reaction_received_type_daily(chat_id, user_id, day_date);

CREATE TABLE IF NOT EXISTS message_author_state (
    chat_id INTEGER NOT NULL,
    message_id INTEGER NOT NULL,
    author_user_id INTEGER NOT NULL,
    author_name TEXT NOT NULL,
    updated_at INTEGER NOT NULL,
    PRIMARY KEY(chat_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_message_author_state_chat_msg ON message_author_state(chat_id, message_id);
