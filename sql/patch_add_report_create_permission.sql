-- 说明：
-- 1) 当前以 sql/init_db.sql 作为唯一主初始化脚本。
-- 2) 本文件仅用于已有库增量升级兼容。
-- 3) 新环境请直接执行 sql/init_db.sql。

INSERT INTO permissions (name, description)
VALUES ('report:create', '批量或补录上报')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name = 'report:create'
WHERE r.name = 'operator'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- super_admin 若通过「全权限」逻辑维护可跳过；若需显式补链可执行：
-- INSERT INTO role_permissions (role_id, permission_id)
-- SELECT r.id, p.id FROM roles r JOIN permissions p ON p.name = 'report:create' WHERE r.name = 'super_admin'
-- ON CONFLICT DO NOTHING;
