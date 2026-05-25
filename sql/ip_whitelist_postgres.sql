-- IP 白名单审计表（可选；应用当前以内存 + Redis 为主）
CREATE TABLE IF NOT EXISTS ip_whitelist_audit (
    id         BIGSERIAL PRIMARY KEY,
    ip         INET NOT NULL,
    remark     VARCHAR(255) NOT NULL DEFAULT '',
    source     VARCHAR(64) NOT NULL DEFAULT 'manual',
    added_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ip_whitelist_audit_ip ON ip_whitelist_audit (ip);
