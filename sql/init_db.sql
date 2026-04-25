-- 初始化 ad_platform 数据库
-- 执行说明：
-- 1. 使用 psql 手动执行以下步骤：
--    a. 连接到 postgres 数据库：psql -U postgres
--    b. 删除已存在的数据库：DROP DATABASE IF EXISTS ad_platform;
--    c. 创建新数据库：CREATE DATABASE ad_platform;
--    d. 连接到新数据库：\c ad_platform;
--    e. 执行此脚本：\i e:\GoPlatform\sql\init_db.sql
-- 注意：每次执行此脚本都会重建整个数据库，确保所有表和数据都是全新的
-- 这样可以保证 ID 是连续的，因为序列会从 1 开始
-- 如果直接在已存在的数据库中执行此脚本，先删除所有表
DROP TABLE IF EXISTS ad_reports CASCADE;
DROP TABLE IF EXISTS ad_tasks CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS clients CASCADE;
DROP TABLE IF EXISTS user_status_logs CASCADE;
DROP TABLE IF EXISTS user_registration_audits CASCADE;
DROP TABLE IF EXISTS user_registrations CASCADE;
DROP TABLE IF EXISTS user_api_keys CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS departments CASCADE;
-- 创建部门表
CREATE TABLE IF NOT EXISTS departments (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description VARCHAR(200),
    parent_id INT REFERENCES departments(id) ON DELETE
    SET NULL,
        status BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- 添加表注释
COMMENT ON TABLE departments IS '部门表';
-- 添加列注释
COMMENT ON COLUMN departments.id IS '部门ID';
COMMENT ON COLUMN departments.name IS '部门名称';
COMMENT ON COLUMN departments.description IS '部门描述';
COMMENT ON COLUMN departments.parent_id IS '父部门ID';
COMMENT ON COLUMN departments.status IS '部门状态';
COMMENT ON COLUMN departments.created_at IS '创建时间';
COMMENT ON COLUMN departments.updated_at IS '更新时间';
-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(100),
    phone VARCHAR(20),
    department_id INT REFERENCES departments(id) ON DELETE
    SET NULL,
        timezone VARCHAR(50) DEFAULT 'Asia/Shanghai',
        status BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- 添加表注释
COMMENT ON TABLE users IS '用户表';
-- 添加列注释
COMMENT ON COLUMN users.id IS '用户ID';
COMMENT ON COLUMN users.username IS '用户名';
COMMENT ON COLUMN users.password IS '密码';
COMMENT ON COLUMN users.email IS '邮箱';
COMMENT ON COLUMN users.name IS '姓名';
COMMENT ON COLUMN users.phone IS '电话';
COMMENT ON COLUMN users.department_id IS '部门ID';
COMMENT ON COLUMN users.timezone IS '时区';
COMMENT ON COLUMN users.status IS '状态';
COMMENT ON COLUMN users.created_at IS '创建时间';
COMMENT ON COLUMN users.updated_at IS '更新时间';
-- 确保 department_id 列存在（如果表已存在但没有该列）
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'users'
        AND column_name = 'department_id'
) THEN
ALTER TABLE users
ADD COLUMN department_id INT REFERENCES departments(id) ON DELETE
SET NULL;
END IF;
END $$;
-- 确保 timezone 列存在（如果表已存在但没有该列）
DO $$ BEGIN IF NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'users'
        AND column_name = 'timezone'
) THEN
ALTER TABLE users
ADD COLUMN timezone VARCHAR(50) DEFAULT 'Asia/Shanghai';
END IF;
END $$;
-- 创建角色表
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- 添加表注释
COMMENT ON TABLE roles IS '角色表';
-- 添加列注释
COMMENT ON COLUMN roles.id IS '角色ID';
COMMENT ON COLUMN roles.name IS '角色名称';
COMMENT ON COLUMN roles.description IS '角色描述';
COMMENT ON COLUMN roles.created_at IS '创建时间';
COMMENT ON COLUMN roles.updated_at IS '更新时间';
-- 创建权限表
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description VARCHAR(200),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- 添加表注释
COMMENT ON TABLE permissions IS '权限表';
-- 添加列注释
COMMENT ON COLUMN permissions.id IS '权限ID';
COMMENT ON COLUMN permissions.name IS '权限名称';
COMMENT ON COLUMN permissions.description IS '权限描述';
COMMENT ON COLUMN permissions.created_at IS '创建时间';
COMMENT ON COLUMN permissions.updated_at IS '更新时间';
-- 创建用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
-- 添加表注释
COMMENT ON TABLE user_roles IS '用户角色关联表';
-- 添加列注释
COMMENT ON COLUMN user_roles.user_id IS '用户ID';
COMMENT ON COLUMN user_roles.role_id IS '角色ID';
-- 创建用户状态变更日志表
CREATE TABLE IF NOT EXISTS user_status_logs (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    operator_id INT REFERENCES users(id) ON DELETE
    SET NULL,
        old_status BOOLEAN NOT NULL,
        new_status BOOLEAN NOT NULL,
        ip VARCHAR(50),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- 添加表注释
COMMENT ON TABLE user_status_logs IS '用户状态变更日志表';
-- 添加列注释
COMMENT ON COLUMN user_status_logs.id IS '日志ID';
COMMENT ON COLUMN user_status_logs.user_id IS '用户ID';
COMMENT ON COLUMN user_status_logs.operator_id IS '操作人ID';
COMMENT ON COLUMN user_status_logs.old_status IS '旧状态';
COMMENT ON COLUMN user_status_logs.new_status IS '新状态';
COMMENT ON COLUMN user_status_logs.ip IS '操作IP';
COMMENT ON COLUMN user_status_logs.created_at IS '创建时间';
-- 创建索引
CREATE INDEX IF NOT EXISTS idx_user_status_logs_user_id ON user_status_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_user_status_logs_operator_id ON user_status_logs(operator_id);
CREATE INDEX IF NOT EXISTS idx_user_status_logs_created_at ON user_status_logs(created_at);
-- 创建用户注册申请表
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
COMMENT ON TABLE user_registrations IS '用户注册申请表';
COMMENT ON COLUMN user_registrations.id IS '注册申请ID';
COMMENT ON COLUMN user_registrations.username IS '用户名';
COMMENT ON COLUMN user_registrations.email IS '邮箱';
COMMENT ON COLUMN user_registrations.status IS '审核状态';
COMMENT ON COLUMN user_registrations.reason IS '审核说明';
COMMENT ON COLUMN user_registrations.reviewed_by IS '审核人ID';
COMMENT ON COLUMN user_registrations.reviewed_at IS '审核时间';
COMMENT ON COLUMN user_registrations.created_at IS '创建时间';
COMMENT ON COLUMN user_registrations.updated_at IS '更新时间';
CREATE INDEX IF NOT EXISTS idx_user_registrations_status ON user_registrations(status);
CREATE INDEX IF NOT EXISTS idx_user_registrations_created_at ON user_registrations(created_at);
-- 创建注册审核日志表
CREATE TABLE IF NOT EXISTS user_registration_audits (
    id SERIAL PRIMARY KEY,
    registration_id INT NOT NULL REFERENCES user_registrations(id) ON DELETE CASCADE,
    operator_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    action VARCHAR(20) NOT NULL,
    reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON TABLE user_registration_audits IS '注册审核日志表';
COMMENT ON COLUMN user_registration_audits.id IS '审核日志ID';
COMMENT ON COLUMN user_registration_audits.registration_id IS '注册申请ID';
COMMENT ON COLUMN user_registration_audits.operator_id IS '操作人ID';
COMMENT ON COLUMN user_registration_audits.action IS '审核动作';
COMMENT ON COLUMN user_registration_audits.reason IS '审核说明';
COMMENT ON COLUMN user_registration_audits.created_at IS '创建时间';
CREATE INDEX IF NOT EXISTS idx_user_registration_audits_registration_id ON user_registration_audits(registration_id);
CREATE INDEX IF NOT EXISTS idx_user_registration_audits_operator_id ON user_registration_audits(operator_id);
-- 创建用户 API Key 表
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
COMMENT ON TABLE user_api_keys IS '用户API Key表';
COMMENT ON COLUMN user_api_keys.id IS '主键ID';
COMMENT ON COLUMN user_api_keys.user_id IS '用户ID';
COMMENT ON COLUMN user_api_keys.key_hash IS 'API Key哈希';
COMMENT ON COLUMN user_api_keys.name IS '密钥名称';
COMMENT ON COLUMN user_api_keys.scope IS '权限范围';
COMMENT ON COLUMN user_api_keys.status IS '状态';
COMMENT ON COLUMN user_api_keys.last_used_at IS '最后使用时间';
COMMENT ON COLUMN user_api_keys.last_used_ip IS '最后使用IP';
COMMENT ON COLUMN user_api_keys.expires_at IS '过期时间';
COMMENT ON COLUMN user_api_keys.created_at IS '创建时间';
COMMENT ON COLUMN user_api_keys.updated_at IS '更新时间';
CREATE INDEX IF NOT EXISTS idx_user_api_keys_user_id ON user_api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_user_api_keys_last_used_at ON user_api_keys(last_used_at);
-- 创建客户表
CREATE TABLE IF NOT EXISTS clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) UNIQUE NOT NULL,
    contact VARCHAR(100),
    phone VARCHAR(20),
    email VARCHAR(100),
    department_id INT REFERENCES departments(id) ON DELETE
    SET NULL,
        salesperson_id INT REFERENCES users(id) ON DELETE
    SET NULL,
        status BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- 添加表注释
COMMENT ON TABLE clients IS '客户表';
-- 添加列注释
COMMENT ON COLUMN clients.id IS '客户ID';
COMMENT ON COLUMN clients.name IS '客户名称';
COMMENT ON COLUMN clients.code IS '客户编码';
COMMENT ON COLUMN clients.contact IS '联系人';
COMMENT ON COLUMN clients.phone IS '联系电话';
COMMENT ON COLUMN clients.email IS '邮箱';
COMMENT ON COLUMN clients.department_id IS '所属部门ID';
COMMENT ON COLUMN clients.salesperson_id IS '销售负责人ID';
COMMENT ON COLUMN clients.status IS '状态';
COMMENT ON COLUMN clients.created_at IS '创建时间';
COMMENT ON COLUMN clients.updated_at IS '更新时间';
-- 创建角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INT REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INT REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);
-- 添加表注释
COMMENT ON TABLE role_permissions IS '角色权限关联表';
-- 添加列注释
COMMENT ON COLUMN role_permissions.role_id IS '角色ID';
COMMENT ON COLUMN role_permissions.permission_id IS '权限ID';
-- 创建广告任务表
CREATE TABLE IF NOT EXISTS ad_tasks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    client_id INT REFERENCES clients(id),
    status VARCHAR(20) DEFAULT 'active',
    priority INTEGER DEFAULT 0,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    target_audience JSONB,
    -- 使用JSONB类型提高查询性能
    creative JSONB,
    -- 使用JSONB类型提高查询性能
    budget DECIMAL(10, 2),
    proxy_strategy INTEGER DEFAULT 1,
    biz_type INTEGER DEFAULT 1,
    profile VARCHAR(50) DEFAULT 'mobile',
    plat VARCHAR(50) DEFAULT 'googlead',
    cau INTEGER DEFAULT 0,
    os INTEGER DEFAULT 1,
    intercept_info JSONB,
    -- 使用JSONB类型提高查询性能
    landurl TEXT,
    allow_county JSONB,
    -- 使用JSONB类型提高查询性能
    final_domains JSONB,
    -- 使用JSONB类型提高查询性能
    info JSONB,
    -- 使用JSONB类型提高查询性能
    user_quality JSONB,
    -- 使用JSONB类型提高查询性能
    click_home_offset_x INTEGER DEFAULT 100,
    click_home_offset_y INTEGER DEFAULT 100,
    click_result_offset_x INTEGER DEFAULT 100,
    click_result_offset_y INTEGER DEFAULT 100,
    f_key_api TEXT,
    plan_id INTEGER,
    -- 执行计划ID
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- 高并发查询优化字段
    last_accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    -- 最后访问时间
    access_count INTEGER DEFAULT 0 -- 访问次数
);
-- 添加表注释
COMMENT ON TABLE ad_tasks IS '广告任务表';
-- 添加列注释
COMMENT ON COLUMN ad_tasks.id IS '任务ID';
COMMENT ON COLUMN ad_tasks.name IS '任务名称';
COMMENT ON COLUMN ad_tasks.description IS '任务描述';
COMMENT ON COLUMN ad_tasks.client_id IS '客户ID';
COMMENT ON COLUMN ad_tasks.status IS '任务状态';
COMMENT ON COLUMN ad_tasks.priority IS '优先级';
COMMENT ON COLUMN ad_tasks.start_time IS '开始时间';
COMMENT ON COLUMN ad_tasks.end_time IS '结束时间';
COMMENT ON COLUMN ad_tasks.target_audience IS '目标受众';
COMMENT ON COLUMN ad_tasks.creative IS '创意';
COMMENT ON COLUMN ad_tasks.budget IS '预算';
COMMENT ON COLUMN ad_tasks.proxy_strategy IS '代理策略';
COMMENT ON COLUMN ad_tasks.biz_type IS '业务类型';
COMMENT ON COLUMN ad_tasks.profile IS '设备类型';
COMMENT ON COLUMN ad_tasks.plat IS '平台';
COMMENT ON COLUMN ad_tasks.cau IS 'CAU值';
COMMENT ON COLUMN ad_tasks.os IS '操作系统';
COMMENT ON COLUMN ad_tasks.intercept_info IS '拦截信息';
COMMENT ON COLUMN ad_tasks.landurl IS '落地页URL';
COMMENT ON COLUMN ad_tasks.allow_county IS '允许的国家';
COMMENT ON COLUMN ad_tasks.final_domains IS '最终域名';
COMMENT ON COLUMN ad_tasks.info IS '附加信息';
COMMENT ON COLUMN ad_tasks.user_quality IS '用户质量';
COMMENT ON COLUMN ad_tasks.click_home_offset_x IS '首页点击偏移X';
COMMENT ON COLUMN ad_tasks.click_home_offset_y IS '首页点击偏移Y';
COMMENT ON COLUMN ad_tasks.click_result_offset_x IS '结果页点击偏移X';
COMMENT ON COLUMN ad_tasks.click_result_offset_y IS '结果页点击偏移Y';
COMMENT ON COLUMN ad_tasks.f_key_api IS 'API密钥';
COMMENT ON COLUMN ad_tasks.plan_id IS '执行计划ID';
COMMENT ON COLUMN ad_tasks.created_at IS '创建时间';
COMMENT ON COLUMN ad_tasks.updated_at IS '更新时间';
COMMENT ON COLUMN ad_tasks.last_accessed_at IS '最后访问时间';
COMMENT ON COLUMN ad_tasks.access_count IS '访问次数';
-- 创建广告上报日志表（分区表）
CREATE TABLE IF NOT EXISTS ad_reports (
    id BIGSERIAL,
    ad_id INT,
    -- 移除外键约束，减少写入开销
    user_id VARCHAR(50),
    action VARCHAR(20) NOT NULL,
    device_info JSONB,
    -- 使用JSONB类型提高查询性能
    ip VARCHAR(50),
    client_inner VARCHAR(50),
    client_outer VARCHAR(50),
    real_ip VARCHAR(50),
    click INTEGER DEFAULT 0,
    use_time DECIMAL(10, 3),
    ads INTEGER DEFAULT 0,
    home_page BOOLEAN DEFAULT false,
    subpage_access INTEGER DEFAULT 0,
    referer TEXT,
    country VARCHAR(10),
    seq VARCHAR(50),
    error TEXT,
    should_click BOOLEAN DEFAULT false,
    should_subpage BOOLEAN DEFAULT false,
    ios BOOLEAN DEFAULT false,
    click_able BOOLEAN DEFAULT false,
    source_type VARCHAR(50),
    ip_source INTEGER DEFAULT 0,
    ad_type INTEGER DEFAULT 0,
    is_review INTEGER DEFAULT 0,
    last_seq VARCHAR(50),
    screen_width INTEGER DEFAULT 0,
    real_url TEXT,
    origin_url TEXT,
    is_save INTEGER DEFAULT 0,
    gpu_error INTEGER DEFAULT 0,
    req_ads_count INTEGER DEFAULT 0,
    match_count INTEGER DEFAULT 0,
    active_view INTEGER DEFAULT 0,
    send_bytes VARCHAR(50) DEFAULT '0',
    recv_bytes VARCHAR(50) DEFAULT '0',
    sleep_info TEXT,
    start_time BIGINT,
    end_time BIGINT,
    plan_id INTEGER,
    time VARCHAR(20),
    -- 当前时间，使用 UTC 标准时间
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
) PARTITION BY RANGE (timestamp);
-- 添加表注释
COMMENT ON TABLE ad_reports IS '广告上报日志表';
-- 添加列注释
COMMENT ON COLUMN ad_reports.id IS '日志ID';
COMMENT ON COLUMN ad_reports.ad_id IS '广告ID';
COMMENT ON COLUMN ad_reports.user_id IS '用户ID';
COMMENT ON COLUMN ad_reports.action IS '操作类型';
COMMENT ON COLUMN ad_reports.device_info IS '设备信息';
COMMENT ON COLUMN ad_reports.ip IS 'IP地址';
COMMENT ON COLUMN ad_reports.client_inner IS '内部客户端';
COMMENT ON COLUMN ad_reports.client_outer IS '外部客户端';
COMMENT ON COLUMN ad_reports.real_ip IS '真实IP';
COMMENT ON COLUMN ad_reports.click IS '点击数';
COMMENT ON COLUMN ad_reports.use_time IS '使用时间';
COMMENT ON COLUMN ad_reports.ads IS '广告数';
COMMENT ON COLUMN ad_reports.home_page IS '是否访问首页';
COMMENT ON COLUMN ad_reports.subpage_access IS '子页面访问次数';
COMMENT ON COLUMN ad_reports.referer IS '来源URL';
COMMENT ON COLUMN ad_reports.country IS '国家';
COMMENT ON COLUMN ad_reports.seq IS '序列';
COMMENT ON COLUMN ad_reports.error IS '错误信息';
COMMENT ON COLUMN ad_reports.should_click IS '是否应点击';
COMMENT ON COLUMN ad_reports.should_subpage IS '是否应访问子页面';
COMMENT ON COLUMN ad_reports.ios IS '是否iOS设备';
COMMENT ON COLUMN ad_reports.click_able IS '是否可点击';
COMMENT ON COLUMN ad_reports.source_type IS '来源类型';
COMMENT ON COLUMN ad_reports.ip_source IS 'IP来源';
COMMENT ON COLUMN ad_reports.ad_type IS '广告类型';
COMMENT ON COLUMN ad_reports.is_review IS '是否审核';
COMMENT ON COLUMN ad_reports.last_seq IS '上一个序列';
COMMENT ON COLUMN ad_reports.screen_width IS '屏幕宽度';
COMMENT ON COLUMN ad_reports.real_url IS '真实URL';
COMMENT ON COLUMN ad_reports.origin_url IS '原始URL';
COMMENT ON COLUMN ad_reports.is_save IS '是否保存';
COMMENT ON COLUMN ad_reports.gpu_error IS 'GPU错误';
COMMENT ON COLUMN ad_reports.req_ads_count IS '请求广告数';
COMMENT ON COLUMN ad_reports.match_count IS '匹配数';
COMMENT ON COLUMN ad_reports.active_view IS '有效视图';
COMMENT ON COLUMN ad_reports.send_bytes IS '发送字节数';
COMMENT ON COLUMN ad_reports.recv_bytes IS '接收字节数';
COMMENT ON COLUMN ad_reports.sleep_info IS '睡眠信息';
COMMENT ON COLUMN ad_reports.start_time IS '开始时间戳';
COMMENT ON COLUMN ad_reports.end_time IS '结束时间戳';
COMMENT ON COLUMN ad_reports.plan_id IS '计划ID';
COMMENT ON COLUMN ad_reports.time IS '当前时间';
COMMENT ON COLUMN ad_reports.timestamp IS '时间戳';
-- 创建主键
ALTER TABLE ad_reports
ADD PRIMARY KEY (id, timestamp);
-- 创建分区（按天分区，提高高并发写入性能）
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d01 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-01') TO ('2026-03-02');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d02 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-02') TO ('2026-03-03');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d03 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-03') TO ('2026-03-04');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d04 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-04') TO ('2026-03-05');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d05 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-05') TO ('2026-03-06');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d06 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-06') TO ('2026-03-07');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d07 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-07') TO ('2026-03-08');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d08 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-08') TO ('2026-03-09');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d09 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-09') TO ('2026-03-10');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d10 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-10') TO ('2026-03-11');
-- 预先创建未来的分区
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d11 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-11') TO ('2026-03-12');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d12 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-12') TO ('2026-03-13');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d13 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-13') TO ('2026-03-14');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d14 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-14') TO ('2026-03-15');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d15 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-15') TO ('2026-03-16');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d16 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-16') TO ('2026-03-17');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d17 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-17') TO ('2026-03-18');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d18 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-18') TO ('2026-03-19');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d19 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-19') TO ('2026-03-20');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d20 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-20') TO ('2026-03-21');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d21 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-21') TO ('2026-03-22');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d22 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-22') TO ('2026-03-23');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d23 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-23') TO ('2026-03-24');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d24 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-24') TO ('2026-03-25');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d25 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-25') TO ('2026-03-26');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d26 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-26') TO ('2026-03-27');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d27 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-27') TO ('2026-03-28');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d28 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-28') TO ('2026-03-29');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d29 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-29') TO ('2026-03-30');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d30 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-30') TO ('2026-03-31');
CREATE TABLE IF NOT EXISTS ad_reports_y2026m03d31 PARTITION OF ad_reports FOR
VALUES
FROM ('2026-03-31') TO ('2026-04-01');
-- 创建索引
-- 广告任务表索引（高并发优化）
CREATE INDEX IF NOT EXISTS idx_ad_tasks_status ON ad_tasks(status);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_priority ON ad_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_plan_id ON ad_tasks(plan_id);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_created_at ON ad_tasks(created_at);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_last_accessed ON ad_tasks(last_accessed_at);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_access_count ON ad_tasks(access_count);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_client_id ON ad_tasks(client_id);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_plat ON ad_tasks(plat);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_os ON ad_tasks(os);
-- 复合索引（针对常见查询模式）
CREATE INDEX IF NOT EXISTS idx_ad_tasks_status_priority ON ad_tasks(status, priority);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_client_id_status ON ad_tasks(client_id, status);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_plat_os ON ad_tasks(plat, os);
CREATE INDEX IF NOT EXISTS idx_ad_tasks_status_created_at ON ad_tasks(status, created_at);
-- 创建物化视图（高并发查询优化）
CREATE MATERIALIZED VIEW mv_active_ad_tasks AS
SELECT id,
    name,
    client_id,
    status,
    priority,
    plat,
    os,
    landurl,
    plan_id,
    info,
    user_quality,
    intercept_info,
    click_home_offset_x,
    click_home_offset_y,
    click_result_offset_x,
    click_result_offset_y,
    f_key_api,
    created_at,
    last_accessed_at,
    access_count
FROM ad_tasks
WHERE status = 'active'
ORDER BY priority DESC,
    created_at DESC;
-- 为物化视图创建索引
CREATE INDEX IF NOT EXISTS idx_mv_active_tasks_priority ON mv_active_ad_tasks(priority);
CREATE INDEX IF NOT EXISTS idx_mv_active_tasks_plat_os ON mv_active_ad_tasks(plat, os);
CREATE INDEX IF NOT EXISTS idx_mv_active_tasks_last_accessed ON mv_active_ad_tasks(last_accessed_at);
-- 创建刷新物化视图的函数
CREATE OR REPLACE FUNCTION refresh_active_tasks_mv() RETURNS void AS $$ BEGIN REFRESH MATERIALIZED VIEW CONCURRENTLY mv_active_ad_tasks;
END;
$$ LANGUAGE plpgsql;
-- 广告上报日志表索引（优化高并发写入）
-- 只保留必要的索引，减少写入开销
CREATE INDEX IF NOT EXISTS idx_ad_reports_seq ON ad_reports(seq);
CREATE INDEX IF NOT EXISTS idx_ad_reports_ad_id ON ad_reports(ad_id);
CREATE INDEX IF NOT EXISTS idx_ad_reports_plan_id ON ad_reports(plan_id);
-- 单独的timestamp索引，优化按时间排序的查询
CREATE INDEX IF NOT EXISTS idx_ad_reports_timestamp ON ad_reports(timestamp);
-- 复合索引（针对常见查询模式）
CREATE INDEX IF NOT EXISTS idx_ad_reports_ad_id_timestamp ON ad_reports(ad_id, timestamp);
-- 分区级索引（在每个分区上创建）
-- 注意：PostgreSQL会自动为分区创建相应的索引
-- 创建批量插入广告上报数据的函数
CREATE OR REPLACE FUNCTION batch_insert_ad_reports(reports JSONB []) RETURNS INTEGER AS $$
DECLARE inserted_count INTEGER := 0;
BEGIN -- 使用UNNEST批量处理
INSERT INTO ad_reports (
        ad_id,
        user_id,
        action,
        device_info,
        ip,
        client_inner,
        client_outer,
        real_ip,
        click,
        use_time,
        ads,
        home_page,
        subpage_access,
        referer,
        country,
        seq,
        error,
        should_click,
        should_subpage,
        ios,
        click_able,
        source_type,
        ip_source,
        ad_type,
        is_review,
        last_seq,
        screen_width,
        real_url,
        origin_url,
        is_save,
        gpu_error,
        req_ads_count,
        match_count,
        active_view,
        send_bytes,
        recv_bytes,
        sleep_info,
        start_time,
        end_time,
        plan_id,
        time,
        timestamp
    )
SELECT r->>'ad_id'::INT,
    r->>'user_id',
    r->>'action',
    r->'device_info',
    r->>'ip',
    r->>'client_inner',
    r->>'client_outer',
    r->>'real_ip',
    (r->>'click')::INTEGER,
    (r->>'use_time')::DECIMAL(10, 3),
    (r->>'ads')::INTEGER,
    (r->>'home_page')::BOOLEAN,
    (r->>'subpage_access')::INTEGER,
    r->>'referer',
    r->>'country',
    r->>'seq',
    r->>'error',
    (r->>'should_click')::BOOLEAN,
    (r->>'should_subpage')::BOOLEAN,
    (r->>'ios')::BOOLEAN,
    (r->>'click_able')::BOOLEAN,
    r->>'source_type',
    (r->>'ip_source')::INTEGER,
    (r->>'ad_type')::INTEGER,
    (r->>'is_review')::INTEGER,
    r->>'last_seq',
    (r->>'screen_width')::INTEGER,
    r->>'real_url',
    r->>'origin_url',
    (r->>'is_save')::INTEGER,
    (r->>'gpu_error')::INTEGER,
    (r->>'req_ads_count')::INTEGER,
    (r->>'match_count')::INTEGER,
    (r->>'active_view')::INTEGER,
    r->>'send_bytes',
    r->>'recv_bytes',
    r->>'sleep_info',
    (r->>'start_time')::BIGINT,
    (r->>'end_time')::BIGINT,
    (r->>'plan_id')::INTEGER,
    r->>'time',
    (r->>'timestamp')::TIMESTAMP
FROM UNNEST(reports) r
RETURNING 1 INTO inserted_count;
RETURN inserted_count;
END;
$$ LANGUAGE plpgsql;
-- 创建自动创建未来月份分区的存储过程
CREATE OR REPLACE FUNCTION create_future_month_partitions(
        p_start_month DATE DEFAULT CURRENT_DATE,
        p_months_to_create INTEGER DEFAULT 6
    ) RETURNS INTEGER AS $$
DECLARE v_current_date DATE;
v_month_start DATE;
v_month_end DATE;
v_partition_name TEXT;
v_month_counter INTEGER := 0;
v_created_count INTEGER := 0;
BEGIN -- 计算起始月份的第一天
v_current_date := DATE_TRUNC('month', p_start_month);
-- 循环创建未来月份的分区
WHILE v_month_counter < p_months_to_create LOOP -- 计算当前月份的起始和结束日期
v_month_start := v_current_date + (v_month_counter || ' months')::INTERVAL;
v_month_end := v_month_start + INTERVAL '1 month';
-- 生成分区表名称
v_partition_name := 'ad_reports_y' || TO_CHAR(v_month_start, 'YYYY') || 'm' || TO_CHAR(v_month_start, 'MM');
-- 检查分区是否已存在
IF NOT EXISTS (
    SELECT 1
    FROM pg_class
    WHERE relname = v_partition_name
) THEN -- 创建月份分区
EXECUTE format(
    'CREATE TABLE IF NOT EXISTS %I PARTITION OF ad_reports FOR VALUES FROM (''%s'') TO (''%s'')',
    v_partition_name,
    v_month_start,
    v_month_end
);
v_created_count := v_created_count + 1;
END IF;
v_month_counter := v_month_counter + 1;
END LOOP;
RETURN v_created_count;
END;
$$ LANGUAGE plpgsql;
-- 插入初始权限数据
INSERT INTO permissions (name, description)
VALUES ('user:create', '创建用户'),
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
    ('report:create', '批量或补录上报') ON CONFLICT (name) DO NOTHING;
-- 插入初始角色数据
INSERT INTO roles (name, description)
VALUES ('super_admin', '超级管理员'),
    ('admin', '管理员'),
    ('operator', '操作员'),
    ('salesperson', '销售人员'),
    ('customer', '客户') ON CONFLICT (name) DO NOTHING;
-- 为角色分配权限
-- 先获取权限ID
WITH perm_ids AS (
    SELECT id,
        name
    FROM permissions
),
role_ids AS (
    SELECT id,
        name
    FROM roles
) -- 插入 role_permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id,
    p.id
FROM role_ids r,
    perm_ids p
WHERE (
        -- super_admin 角色 (所有权限)
        (r.name = 'super_admin')
        OR -- admin 角色 (只读权限)
        (
            r.name = 'admin'
            AND p.name IN (
                'user:read',
                'role:read',
                'department:read',
                'ad:read',
                'client:read',
                'report:read'
            )
        )
        OR -- operator 角色 (部门内管理权限)
        (
            r.name = 'operator'
            AND p.name IN (
                'user:read',
                'user:update',
                'department:read',
                'department:update',
                'ad:read',
                'ad:update',
                'client:read',
                'client:update',
                'report:read',
                'report:create'
            )
        )
        OR -- salesperson 角色 (个人客户管理权限)
        (
            r.name = 'salesperson'
            AND p.name IN ('client:read', 'client:update', 'report:read')
        )
        OR -- customer 角色 (个人信息查看权限)
        (
            r.name = 'customer'
            AND p.name IN ('client:read', 'report:read')
        )
    ) ON CONFLICT (role_id, permission_id) DO NOTHING;
-- 创建默认超级管理员用户
INSERT INTO "public"."users" (
        "username",
        "password",
        "email",
        "name",
        "phone",
        "department_id",
        "timezone",
        "status"
    )
VALUES (
        'super_admin',
        '$2a$10$dhd0M.R58/Uebemq5toLFe2c6422rZs5CmS52dj3jyoJm7FbLOMKy',
        'admin@example.com',
        '超级管理员',
        '13000000001',
        NULL,
        'Asia/Shanghai',
        't'
    );

-- 为超级管理员分配角色
WITH admin_user AS (
    SELECT id
    FROM users
    WHERE username = 'super_admin'
),
super_admin_role AS (
    SELECT id
    FROM roles
    WHERE name = 'super_admin'
)
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM admin_user u,
    super_admin_role r ON CONFLICT (user_id, role_id) DO NOTHING;
-- 插入初始部门数据
INSERT INTO departments (name, description, status)
VALUES 
    ('总部', '公司总部', true),
    ('技术部', '技术开发部门', true),
    ('市场部', '市场营销部门', true),
    ('销售部', '销售业务部门', true),
    ('财务部', '财务管理部门', true) ON CONFLICT (name) DO NOTHING;