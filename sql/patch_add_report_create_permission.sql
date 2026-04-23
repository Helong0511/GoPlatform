-- 已有库升级：补充 report:create 权限，并授予 operator（与 init_db 一致）
-- 执行：在 ad_platform 库中 psql -f sql/patch_add_report_create_permission.sql

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
