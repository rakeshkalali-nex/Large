CREATE TABLE settings_history (
 version INTEGER PRIMARY KEY,
 snapshot_json TEXT NOT NULL CHECK(json_valid(snapshot_json)),
 actor_id TEXT NOT NULL REFERENCES users(id),
 occurred_at TEXT NOT NULL
) STRICT;
CREATE TRIGGER settings_history_no_update BEFORE UPDATE ON settings_history BEGIN SELECT RAISE(ABORT,'immutable settings history'); END;
CREATE TRIGGER settings_history_no_delete BEFORE DELETE ON settings_history BEGIN SELECT RAISE(ABORT,'immutable settings history'); END;
