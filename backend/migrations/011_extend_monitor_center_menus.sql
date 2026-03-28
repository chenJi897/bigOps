-- 011_extend_monitor_center_menus.sql
-- 监控中心扩展菜单：告警事件 / 监控数据源 / PromQL 查询台

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'alert_events', '告警事件', 'Warning', '/monitor/alerts', 'AlertEvents', '/api/v1/alert-events*', '*', 2, 3, 1, 1
FROM `menus` m WHERE m.`name` = 'monitor_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = VALUES(`title`),
  `icon` = VALUES(`icon`),
  `path` = VALUES(`path`),
  `component` = VALUES(`component`),
  `api_path` = VALUES(`api_path`),
  `api_method` = VALUES(`api_method`),
  `type` = VALUES(`type`),
  `sort` = VALUES(`sort`),
  `visible` = VALUES(`visible`),
  `status` = VALUES(`status`);

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'monitor_datasources', '监控数据源', 'Connection', '/monitor/datasources', 'MonitorDatasources', '/api/v1/monitor/datasources*', '*', 2, 4, 1, 1
FROM `menus` m WHERE m.`name` = 'monitor_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = VALUES(`title`),
  `icon` = VALUES(`icon`),
  `path` = VALUES(`path`),
  `component` = VALUES(`component`),
  `api_path` = VALUES(`api_path`),
  `api_method` = VALUES(`api_method`),
  `type` = VALUES(`type`),
  `sort` = VALUES(`sort`),
  `visible` = VALUES(`visible`),
  `status` = VALUES(`status`);

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'monitor_query', 'PromQL 查询台', 'DataLine', '/monitor/query', 'MonitorQuery', '/api/v1/monitor/query*', '*', 2, 5, 1, 1
FROM `menus` m WHERE m.`name` = 'monitor_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = VALUES(`title`),
  `icon` = VALUES(`icon`),
  `path` = VALUES(`path`),
  `component` = VALUES(`component`),
  `api_path` = VALUES(`api_path`),
  `api_method` = VALUES(`api_method`),
  `type` = VALUES(`type`),
  `sort` = VALUES(`sort`),
  `visible` = VALUES(`visible`),
  `status` = VALUES(`status`);

INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
JOIN `menus` m ON m.`name` IN ('monitor_dir', 'monitor_dashboard', 'alert_rules', 'alert_events', 'monitor_datasources', 'monitor_query')
WHERE r.`name` IN ('admin', 'ops')
  AND r.`deleted_at` IS NULL
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
