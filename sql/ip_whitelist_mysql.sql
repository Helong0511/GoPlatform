-- IP 白名单审计表（MySQL 8.0+，可选）
-- 应用当前以内存 + Redis 为主；此表供运维落库或后续扩展
-- 使用：USE ad_platform; SOURCE /path/to/sql/ip_whitelist_mysql.sql;

CREATE TABLE IF NOT EXISTS ip_whitelist_audit (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ip         VARCHAR(45) NOT NULL COMMENT 'IPv4/IPv6 字符串',
    remark     VARCHAR(255) NOT NULL DEFAULT '',
    source     VARCHAR(64) NOT NULL DEFAULT 'manual',
    added_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ip_whitelist_audit_ip (ip)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
