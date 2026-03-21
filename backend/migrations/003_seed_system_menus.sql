-- 003_seed_system_menus.sql
-- 修正系统管理菜单数据，确保路径和组件与前端路由完全对应
-- 如果已执行过 002 的菜单 INSERT，请先执行本文件进行修正

-- 删除 002 中插入的错误菜单（路径不对、缺少 component）
DELETE FROM `menus` WHERE `name` IN ('system', 'user_list', 'role_list', 'menu_list');

-- 系统管理目录（type=1：目录，无路由）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'system_dir', '系统管理', 'Setting', '', '', 1, 100, 1, 1)
ON DUPLICATE KEY UPDATE `title` = '系统管理', `icon` = 'Setting', `path` = '', `component` = '';

-- 用户管理（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'user_management', '用户管理', 'User', '/users', 'Users', '/api/v1/users', 'GET', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'system_dir'
ON DUPLICATE KEY UPDATE `title` = '用户管理', `icon` = 'User', `path` = '/users', `component` = 'Users';

-- 角色管理（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'role_management', '角色管理', 'UserFilled', '/roles', 'Roles', '/api/v1/roles', 'GET', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'system_dir'
ON DUPLICATE KEY UPDATE `title` = '角色管理', `icon` = 'UserFilled', `path` = '/roles', `component` = 'Roles';

-- 菜单管理（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'menu_management', '菜单管理', 'Menu', '/menus', 'Menus', '/api/v1/menus', 'GET', 2, 3, 1, 1
FROM `menus` m WHERE m.`name` = 'system_dir'
ON DUPLICATE KEY UPDATE `title` = '菜单管理', `icon` = 'Menu', `path` = '/menus', `component` = 'Menus';

-- 审计日志（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'audit_logs', '审计日志', 'DocumentChecked', '/audit-logs', 'AuditLogs', '/api/v1/audit-logs', 'GET', 2, 4, 1, 1
FROM `menus` m WHERE m.`name` = 'system_dir'
ON DUPLICATE KEY UPDATE `title` = '审计日志', `icon` = 'DocumentChecked', `path` = '/audit-logs', `component` = 'AuditLogs';

-- 给 admin 角色分配所有系统管理菜单权限
INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
CROSS JOIN `menus` m
WHERE r.`name` = 'admin'
  AND m.`name` IN ('system_dir', 'user_management', 'role_management', 'menu_management', 'audit_logs')
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
