-- IP 黑名单审计表（可选；应用当前以内存 + Redis 为主，此表供运维落库或后续扩展）
CREATE TABLE IF NOT EXISTS ip_blacklist_audit (
    id          BIGSERIAL PRIMARY KEY,
    ip          INET NOT NULL,
    reason      VARCHAR(255) NOT NULL,
    banned_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ,
    source      VARCHAR(64) NOT NULL DEFAULT 'auto',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ip_blacklist_audit_ip ON ip_blacklist_audit (ip);
CREATE INDEX IF NOT EXISTS idx_ip_blacklist_audit_banned_at ON ip_blacklist_audit (banned_at DESC);
