-- 023_menu_notify_groups.sql
-- 发送组菜单挂入监控中心

INSERT INTO menus (name, title, path, component, parent_id, sort, icon, status)
VALUES ('notify_groups', '发送组', '/monitor/notify-groups', 'NotifyGroups',
        (SELECT id FROM (SELECT id FROM menus WHERE name = 'monitor_dir') t), 33, 'Message', 1)
ON DUPLICATE KEY UPDATE title='发送组', path='/monitor/notify-groups', component='NotifyGroups',
    sort=33, icon='Message', status=1;

-- admin 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT 1, id FROM menus WHERE name = 'notify_groups';

-- ops 角色绑定
INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT 4, id FROM menus WHERE name = 'notify_groups';

-- Casbin API 权限
INSERT IGNORE INTO casbin_rule (ptype, v0, v1, v2) VALUES
('p', 'admin', '/api/v1/notify-groups', 'GET'),
('p', 'admin', '/api/v1/notify-groups', 'POST'),
('p', 'admin', '/api/v1/notify-groups/*', 'GET'),
('p', 'admin', '/api/v1/notify-groups/*', 'POST'),
('p', 'ops', '/api/v1/notify-groups', 'GET'),
('p', 'ops', '/api/v1/notify-groups/*', 'GET');
