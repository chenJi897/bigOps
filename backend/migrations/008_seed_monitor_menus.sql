-- 008_seed_monitor_menus.sql
-- 监控模块菜单：监控大盘 / 告警规则

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'monitor_dir', '监控中心', 'DataAnalysis', '/monitor', '', 1, 120, 1, 1)
ON DUPLICATE KEY UPDATE
  `title` = '监控中心',
  `icon` = 'DataAnalysis',
  `path` = '/monitor',
  `component` = '',
  `type` = 1,
  `sort` = 120,
  `visible` = 1,
  `status` = 1;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'monitor_dashboard', '监控大盘', 'Odometer', '/monitor/dashboard', 'MonitorDashboard', '/api/v1/monitor*', '*', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'monitor_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = '监控大盘',
  `icon` = 'Odometer',
  `path` = '/monitor/dashboard',
  `component` = 'MonitorDashboard',
  `api_path` = '/api/v1/monitor*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 1,
  `visible` = 1,
  `status` = 1;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'alert_rules', '告警规则', 'Bell', '/monitor/alert-rules', 'AlertRules', '/api/v1/alert*', '*', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'monitor_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = '告警规则',
  `icon` = 'Bell',
  `path` = '/monitor/alert-rules',
  `component` = 'AlertRules',
  `api_path` = '/api/v1/alert*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 2,
  `visible` = 1,
  `status` = 1;

INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
JOIN `menus` m ON m.`name` IN ('monitor_dir', 'monitor_dashboard', 'alert_rules')
WHERE r.`name` IN ('admin', 'ops')
  AND r.`deleted_at` IS NULL
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
