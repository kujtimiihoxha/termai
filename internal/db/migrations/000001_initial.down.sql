DROP TRIGGER IF EXISTS update_sessions_updated_at;
DROP TRIGGER IF EXISTS update_messages_updated_at;
DROP TRIGGER IF EXISTS update_files_updated_at;

DROP TRIGGER IF EXISTS update_session_message_count_on_delete;
DROP TRIGGER IF EXISTS update_session_message_count_on_insert;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS files;
