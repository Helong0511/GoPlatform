-- IP 黑名单审计表（MySQL 8.0+，可选）
-- 应用当前以内存 + Redis 为主；此表供运维落库或后续扩展
-- 使用：USE ad_platform; SOURCE /path/to/sql/ip_blacklist_mysql.sql;

CREATE TABLE IF NOT EXISTS ip_blacklist_audit (
    id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ip          VARCHAR(45) NOT NULL COMMENT 'IPv4/IPv6 字符串',
    reason      VARCHAR(255) NOT NULL,
    banned_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  TIMESTAMP NULL DEFAULT NULL,
    source      VARCHAR(64) NOT NULL DEFAULT 'auto',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ip_blacklist_audit_ip (ip),
    KEY idx_ip_blacklist_audit_banned_at (banned_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
