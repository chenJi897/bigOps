-- 017_seed_dashboard_menus.sql
-- 添加仪表盘目录及其子菜单（工作台、概览分析）

-- 仪表盘目录（type=1：目录）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'dashboard_dir', '仪表盘', 'Odometer', '', '', 1, 0, 1, 1)
ON DUPLICATE KEY UPDATE `title` = '仪表盘', `icon` = 'Odometer', `path` = '', `component` = '';

-- 工作台（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'dashboard_workbench', '工作台', 'Monitor', '/dashboard/workbench', 'DashboardWorkbench', '/api/v1/stats/*', 'GET', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'dashboard_dir'
ON DUPLICATE KEY UPDATE `title` = '工作台', `icon` = 'Monitor', `path` = '/dashboard/workbench', `component` = 'DashboardWorkbench';

-- 概览分析（type=2：菜单页面）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'dashboard_overview', '概览分析', 'DataAnalysis', '/dashboard/overview', 'DashboardOverview', '/api/v1/stats/*', 'GET', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'dashboard_dir'
ON DUPLICATE KEY UPDATE `title` = '概览分析', `icon` = 'DataAnalysis', `path` = '/dashboard/overview', `component` = 'DashboardOverview';

-- 给 admin 角色分配仪表盘菜单权限
INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
CROSS JOIN `menus` m
WHERE r.`name` = 'admin'
  AND m.`name` IN ('dashboard_dir', 'dashboard_workbench', 'dashboard_overview')
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
