-- MySQL 8.0 初始化脚本（ad_platform）
-- 使用方式：
--   1) mysql -uroot -p
--   2) CREATE DATABASE IF NOT EXISTS ad_platform CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
--   3) USE ad_platform;
--   4) SOURCE /path/to/GoPlatform/sql/init_db_mysql.sql;

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS ip_whitelist_audit;
DROP TABLE IF EXISTS ip_blacklist_audit;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS user_status_logs;
DROP TABLE IF EXISTS user_api_keys;
DROP TABLE IF EXISTS user_registration_audits;
DROP TABLE IF EXISTS user_registrations;
DROP TABLE IF EXISTS ad_reports;
DROP TABLE IF EXISTS ad_tasks;
DROP TABLE IF EXISTS clients;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS departments;

SET FOREIGN_KEY_CHECKS = 1;

-- 部门表（departments）
-- 字段说明：
--   id: 部门主键ID（自增）
--   name: 部门名称（唯一）
--   description: 部门描述
--   parent_id: 父部门ID（支持树状层级）
--   status: 部门状态（1=正常，0=归档）
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE departments (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(200) NULL,
    parent_id BIGINT UNSIGNED NULL,
    status TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_departments_name (name),
    KEY idx_departments_parent_id (parent_id),
    CONSTRAINT fk_departments_parent_id FOREIGN KEY (parent_id) REFERENCES departments(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户表（users）
-- 字段说明：
--   id: 用户主键ID（自增）
--   username: 登录用户名（唯一）
--   password: 登录密码哈希（bcrypt等）
--   email: 用户邮箱（唯一）
--   name: 姓名/显示名
--   phone: 联系电话
--   department_id: 所属部门ID（可空）
--   timezone: 用户时区（默认 Asia/Shanghai）
--   status: 账号状态（1=启用，0=禁用）
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    password VARCHAR(100) NOT NULL,
    email VARCHAR(100) NOT NULL,
    name VARCHAR(100) NULL,
    phone VARCHAR(20) NULL,
    department_id BIGINT UNSIGNED NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Shanghai',
    status TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_users_username (username),
    UNIQUE KEY uk_users_email (email),
    KEY idx_users_department_id (department_id),
    CONSTRAINT fk_users_department_id FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 注册申请表（user_registrations）
-- 字段说明：
--   id: 注册申请ID
--   username: 申请用户名
--   email: 申请邮箱
--   status: 审核状态（pending/approved/rejected）
--   reason: 审核原因/备注
--   reviewed_by: 审核人用户ID
--   reviewed_at: 审核时间
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE user_registrations (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    reason VARCHAR(500) NULL,
    reviewed_by BIGINT UNSIGNED NULL,
    reviewed_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_user_registrations_status (status),
    KEY idx_user_registrations_created_at (created_at),
    KEY idx_user_registrations_reviewed_by (reviewed_by),
    CONSTRAINT fk_user_registrations_reviewed_by FOREIGN KEY (reviewed_by) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 注册审核日志表（user_registration_audits）
-- 字段说明：
--   id: 审核日志ID
--   registration_id: 对应注册申请ID
--   operator_id: 操作人用户ID
--   action: 审核动作（approve/reject等）
--   reason: 审核说明
--   created_at: 日志创建时间
CREATE TABLE user_registration_audits (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    registration_id BIGINT UNSIGNED NOT NULL,
    operator_id BIGINT UNSIGNED NULL,
    action VARCHAR(20) NOT NULL,
    reason VARCHAR(500) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_user_registration_audits_registration_id (registration_id),
    KEY idx_user_registration_audits_operator_id (operator_id),
    CONSTRAINT fk_user_registration_audits_registration_id FOREIGN KEY (registration_id) REFERENCES user_registrations(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_registration_audits_operator_id FOREIGN KEY (operator_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户 API Key 表（user_api_keys）
-- 字段说明：
--   id: 主键ID
--   user_id: 归属用户ID
--   key_hash: API Key 哈希值（唯一）
--   name: 密钥名称（便于管理）
--   scope: 权限范围（默认 all）
--   status: 状态（1=有效，0=停用）
--   last_used_at: 最后使用时间
--   last_used_ip: 最后使用IP
--   expires_at: 过期时间（可空表示不过期）
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE user_api_keys (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    key_hash VARCHAR(255) NOT NULL,
    name VARCHAR(100) NOT NULL,
    scope VARCHAR(200) NOT NULL DEFAULT 'all',
    status TINYINT(1) NOT NULL DEFAULT 1,
    last_used_at DATETIME NULL,
    last_used_ip VARCHAR(50) NULL,
    expires_at DATETIME NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_user_api_keys_key_hash (key_hash),
    KEY idx_user_api_keys_user_id (user_id),
    KEY idx_user_api_keys_last_used_at (last_used_at),
    CONSTRAINT fk_user_api_keys_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色表（roles）
-- 字段说明：
--   id: 角色主键ID
--   name: 角色名称（唯一）
--   description: 角色描述
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE roles (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(200) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_roles_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 权限表（permissions）
-- 字段说明：
--   id: 权限主键ID
--   name: 权限标识（如 user:read，唯一）
--   description: 权限说明
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE permissions (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(200) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_permissions_name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户-角色关联表（user_roles）
-- 字段说明：
--   user_id: 用户ID
--   role_id: 角色ID
-- 复合主键 (user_id, role_id) 防重复分配
CREATE TABLE user_roles (
    user_id BIGINT UNSIGNED NOT NULL,
    role_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (user_id, role_id),
    KEY idx_user_roles_role_id (role_id),
    CONSTRAINT fk_user_roles_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 角色-权限关联表（role_permissions）
-- 字段说明：
--   role_id: 角色ID
--   permission_id: 权限ID
-- 复合主键 (role_id, permission_id) 防重复分配
CREATE TABLE role_permissions (
    role_id BIGINT UNSIGNED NOT NULL,
    permission_id BIGINT UNSIGNED NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    KEY idx_role_permissions_permission_id (permission_id),
    CONSTRAINT fk_role_permissions_role_id FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT fk_role_permissions_permission_id FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 用户状态变更日志表（user_status_logs）
-- 字段说明：
--   id: 日志主键ID
--   user_id: 目标用户ID
--   operator_id: 操作人ID
--   old_status: 旧状态（0/1）
--   new_status: 新状态（0/1）
--   ip: 操作来源IP
--   created_at: 日志创建时间
CREATE TABLE user_status_logs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    operator_id BIGINT UNSIGNED NULL,
    old_status TINYINT(1) NOT NULL,
    new_status TINYINT(1) NOT NULL,
    ip VARCHAR(50) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_user_status_logs_user_id (user_id),
    KEY idx_user_status_logs_operator_id (operator_id),
    KEY idx_user_status_logs_created_at (created_at),
    CONSTRAINT fk_user_status_logs_user_id FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_user_status_logs_operator_id FOREIGN KEY (operator_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 客户表（clients，下游=客户=渠道=商户）
-- 字段说明：
--   id: 客户主键ID
--   name: 客户名称
--   code: 客户编码（唯一）
--   contact: 联系人
--   phone: 联系电话
--   email: 联系邮箱
--   department_id: 所属部门ID
--   salesperson_id: 销售负责人ID
--   status: 状态（1=启用，0=停用）
--   created_at: 创建时间
--   updated_at: 更新时间
CREATE TABLE clients (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL,
    contact VARCHAR(100) NULL,
    phone VARCHAR(20) NULL,
    email VARCHAR(100) NULL,
    department_id BIGINT UNSIGNED NULL,
    salesperson_id BIGINT UNSIGNED NULL,
    status TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_clients_code (code),
    KEY idx_clients_department_id (department_id),
    KEY idx_clients_salesperson_id (salesperson_id),
    CONSTRAINT fk_clients_department_id FOREIGN KEY (department_id) REFERENCES departments(id) ON DELETE SET NULL,
    CONSTRAINT fk_clients_salesperson_id FOREIGN KEY (salesperson_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 广告任务表（ad_tasks，上游=广告主任务）
-- 字段说明：
--   id: 任务主键ID
--   name: 任务名称
--   description: 任务描述
--   client_id: 客户ID
--   status: 任务状态（active/inactive等）
--   priority: 优先级（越大越优先）
--   start_time/end_time: 投放开始/结束时间
--   target_audience: 目标受众(JSON)
--   creative: 创意配置(JSON)
--   budget: 预算
--   proxy_strategy: 代理策略
--   biz_type: 业务类型
--   profile: 设备画像
--   plat: 平台（如 googlead）
--   cau/os: 业务自定义维度
--   intercept_info: 拦截规则(JSON)
--   landurl: 落地页URL
--   allow_county/final_domains/info/user_quality: 业务配置(JSON)
--   click_home_offset_x/y: 首页点击偏移
--   click_result_offset_x/y: 结果页点击偏移
--   f_key_api: 外部API密钥信息（建议加密存储）
--   plan_id: 计划ID
--   created_at/updated_at: 创建/更新时间
--   last_accessed_at: 最近访问时间
--   access_count: 访问次数
CREATE TABLE ad_tasks (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT NULL,
    client_id BIGINT UNSIGNED NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    priority INT NOT NULL DEFAULT 0,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    target_audience JSON NULL,
    creative JSON NULL,
    budget DECIMAL(10,2) NOT NULL DEFAULT 0.00,
    proxy_strategy INT NOT NULL DEFAULT 1,
    biz_type INT NOT NULL DEFAULT 1,
    profile VARCHAR(50) NOT NULL DEFAULT 'mobile',
    plat VARCHAR(50) NOT NULL DEFAULT 'googlead',
    cau INT NOT NULL DEFAULT 0,
    os INT NOT NULL DEFAULT 1,
    intercept_info JSON NULL,
    landurl TEXT NULL,
    allow_county JSON NULL,
    final_domains JSON NULL,
    info JSON NULL,
    user_quality JSON NULL,
    click_home_offset_x INT NOT NULL DEFAULT 100,
    click_home_offset_y INT NOT NULL DEFAULT 100,
    click_result_offset_x INT NOT NULL DEFAULT 100,
    click_result_offset_y INT NOT NULL DEFAULT 100,
    f_key_api TEXT NULL,
    plan_id INT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    access_count INT NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_ad_tasks_client_id (client_id),
    KEY idx_ad_tasks_status (status),
    KEY idx_ad_tasks_priority (priority),
    KEY idx_ad_tasks_created_at (created_at),
    KEY idx_ad_tasks_plan_id (plan_id),
    KEY idx_ad_tasks_plat (plat),
    KEY idx_ad_tasks_os (os),
    KEY idx_ad_tasks_last_accessed_at (last_accessed_at),
    KEY idx_ad_tasks_access_count (access_count),
    CONSTRAINT fk_ad_tasks_client_id FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 广告上报表（ad_reports）
-- 字段说明：
--   id: 上报主键ID
--   ad_id: 广告任务ID
--   user_id: 执行用户标识
--   action: 行为类型（click/impression/conversion等）
--   device_info: 设备信息(JSON)
--   ip/client_inner/client_outer/real_ip: 网络相关字段
--   click/use_time/ads/home_page/subpage_access: 行为计数与耗时
--   referer/country/seq/error: 来源、国家、序列、错误信息
--   should_click/should_subpage/ios/click_able: 执行标志
--   source_type/ip_source/ad_type/is_review: 业务分类字段
--   last_seq/screen_width/real_url/origin_url: 页面与设备补充信息
--   is_save/gpu_error/req_ads_count/match_count/active_view: 质量统计字段
--   send_bytes/recv_bytes/sleep_info: 资源与睡眠信息
--   start_time/end_time: 上报内时间戳（毫秒/秒由业务定义）
--   plan_id/time: 计划ID与业务时间字段
--   timestamp: 服务端入库时间（关键统计索引）
CREATE TABLE ad_reports (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ad_id BIGINT UNSIGNED NULL,
    user_id VARCHAR(50) NULL,
    action VARCHAR(20) NOT NULL,
    device_info JSON NULL,
    ip VARCHAR(50) NULL,
    client_inner VARCHAR(50) NULL,
    client_outer VARCHAR(50) NULL,
    real_ip VARCHAR(50) NULL,
    click INT NOT NULL DEFAULT 0,
    use_time DECIMAL(10,3) NOT NULL DEFAULT 0.000,
    ads INT NOT NULL DEFAULT 0,
    home_page TINYINT(1) NOT NULL DEFAULT 0,
    subpage_access INT NOT NULL DEFAULT 0,
    referer TEXT NULL,
    country VARCHAR(10) NULL,
    seq VARCHAR(50) NULL,
    error TEXT NULL,
    should_click TINYINT(1) NOT NULL DEFAULT 0,
    should_subpage TINYINT(1) NOT NULL DEFAULT 0,
    ios TINYINT(1) NOT NULL DEFAULT 0,
    click_able TINYINT(1) NOT NULL DEFAULT 0,
    source_type VARCHAR(50) NULL,
    ip_source INT NOT NULL DEFAULT 0,
    ad_type INT NOT NULL DEFAULT 0,
    is_review INT NOT NULL DEFAULT 0,
    last_seq VARCHAR(50) NULL,
    screen_width INT NOT NULL DEFAULT 0,
    real_url TEXT NULL,
    origin_url TEXT NULL,
    is_save INT NOT NULL DEFAULT 0,
    gpu_error INT NOT NULL DEFAULT 0,
    req_ads_count INT NOT NULL DEFAULT 0,
    match_count INT NOT NULL DEFAULT 0,
    active_view INT NOT NULL DEFAULT 0,
    send_bytes VARCHAR(50) NOT NULL DEFAULT '0',
    recv_bytes VARCHAR(50) NOT NULL DEFAULT '0',
    sleep_info TEXT NULL,
    start_time BIGINT NULL,
    end_time BIGINT NULL,
    plan_id INT NULL,
    `time` VARCHAR(20) NULL,
    timestamp DATETIME NOT NULL,
    PRIMARY KEY (id),
    KEY idx_ad_reports_ad_id (ad_id),
    KEY idx_ad_reports_action (action),
    KEY idx_ad_reports_timestamp (timestamp)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO permissions (name, description)
VALUES
    ('user:create', '创建用户'),
    ('user:read', '读取用户'),
    ('user:update', '更新用户'),
    ('user:delete', '删除用户'),
    ('role:create', '创建角色'),
    ('role:read', '读取角色'),
    ('role:update', '更新角色'),
    ('role:delete', '删除角色'),
    ('department:create', '创建部门'),
    ('department:read', '读取部门'),
    ('department:update', '更新部门'),
    ('department:delete', '删除部门'),
    ('ad:create', '创建广告'),
    ('ad:read', '读取广告'),
    ('ad:update', '更新广告'),
    ('ad:delete', '删除广告'),
    ('client:create', '创建客户'),
    ('client:read', '读取客户'),
    ('client:update', '更新客户'),
    ('client:delete', '删除客户'),
    ('report:read', '读取报告'),
    ('report:create', '批量或补录上报');

INSERT IGNORE INTO roles (name, description)
VALUES
    ('super_admin', '超级管理员'),
    ('admin', '管理员'),
    ('operator', '操作员'),
    ('salesperson', '销售人员'),
    ('customer', '客户');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p
WHERE
    (r.name = 'super_admin')
    OR (
        r.name = 'admin'
        AND p.name IN ('user:read', 'role:read', 'department:read', 'ad:read', 'client:read', 'report:read')
    )
    OR (
        r.name = 'operator'
        AND p.name IN (
            'user:read', 'user:update',
            'department:read', 'department:update',
            'ad:read', 'ad:update',
            'client:read', 'client:update',
            'report:read', 'report:create'
        )
    )
    OR (
        r.name = 'salesperson'
        AND p.name IN ('client:read', 'client:update', 'report:read')
    )
    OR (
        r.name = 'customer'
        AND p.name IN ('client:read', 'report:read')
    );

-- IP 黑名单审计表（可选；应用以内存 + Redis 为主，见 sql/ip_blacklist_mysql.sql）
CREATE TABLE ip_blacklist_audit (
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

-- IP 白名单审计表（可选；见 sql/ip_whitelist_mysql.sql）
CREATE TABLE ip_whitelist_audit (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    ip         VARCHAR(45) NOT NULL COMMENT 'IPv4/IPv6 字符串',
    remark     VARCHAR(255) NOT NULL DEFAULT '',
    source     VARCHAR(64) NOT NULL DEFAULT 'manual',
    added_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_ip_whitelist_audit_ip (ip)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 可选：默认超级管理员（按需修改密码哈希）
-- INSERT INTO users (username, password, email, name, status)
-- VALUES ('admin', '$2a$10$replace_with_bcrypt_hash', 'admin@example.com', 'Super Admin', 1);

