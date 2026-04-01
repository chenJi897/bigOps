-- 018_seed_personal_center_menus.sql
-- 添加个人中心目录及其子菜单（个人设置、通知偏好）

-- 个人中心目录（type=1：目录，排序靠后）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'personal_center_dir', '个人中心', 'UserFilled', '', '', 1, 90, 1, 1)
ON DUPLICATE KEY UPDATE `title` = '个人中心', `icon` = 'UserFilled', `sort` = 90;

-- 个人设置（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'personal_settings', '个人设置', 'Setting', '/user/settings', 'UserSettings', '/api/v1/auth/*', 'GET', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'personal_center_dir'
ON DUPLICATE KEY UPDATE `title` = '个人设置', `icon` = 'Setting', `path` = '/user/settings', `component` = 'UserSettings';

-- 通知偏好（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'notification_center', '通知配置中心', 'Bell', '/notification/console', 'NotificationConsole', '/api/v1/notifications/*', 'GET', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'personal_center_dir'
ON DUPLICATE KEY UPDATE `title` = '通知配置中心', `icon` = 'Bell', `path` = '/notification/console', `component` = 'NotificationConsole';

-- 给 admin 角色分配权限
INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
CROSS JOIN `menus` m
WHERE r.`name` = 'admin'
  AND m.`name` IN ('personal_center_dir', 'personal_settings', 'notification_center')
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
