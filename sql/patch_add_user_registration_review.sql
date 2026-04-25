-- 说明：
-- 1) 当前以 sql/init_db.sql 作为唯一主初始化脚本。
-- 2) 本文件仅用于已有库增量升级兼容。
-- 3) 新环境请直接执行 sql/init_db.sql。
-- 注册申请与审核留痕
CREATE TABLE IF NOT EXISTS user_registrations (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reason VARCHAR(255),
    reviewed_by INT REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_registrations_status ON user_registrations(status);
CREATE INDEX IF NOT EXISTS idx_user_registrations_created_at ON user_registrations(created_at);

CREATE TABLE IF NOT EXISTS user_registration_audits (
    id SERIAL PRIMARY KEY,
    registration_id INT NOT NULL REFERENCES user_registrations(id) ON DELETE CASCADE,
    operator_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(20) NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_registration_audits_registration_id ON user_registration_audits(registration_id);
CREATE INDEX IF NOT EXISTS idx_user_registration_audits_operator_id ON user_registration_audits(operator_id);

CREATE TABLE IF NOT EXISTS user_api_keys (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(50) DEFAULT 'default',
    scope VARCHAR(200) DEFAULT 'all',
    status BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMP,
    last_used_ip VARCHAR(50),
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_last_used_at ON user_api_keys(last_used_at);

ALTER TABLE user_api_keys ADD COLUMN IF NOT EXISTS scope VARCHAR(200) DEFAULT 'all';
ALTER TABLE user_api_keys ADD COLUMN IF NOT EXISTS last_used_ip VARCHAR(50);
ALTER TABLE user_api_keys ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP;
