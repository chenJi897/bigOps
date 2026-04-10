-- 026_seed_task_executions_visible_menu.sql
-- 任务中心：补充「执行记录」可见菜单，侧栏与 RBAC 菜单树一致
-- 幂等：INSERT ... ON DUPLICATE KEY UPDATE

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_executions', '执行记录', 'Document', '/task/executions', 'TaskExecutions', '/api/v1/task-executions*', '*', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'task_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '执行记录',
  `icon`       = 'Document',
  `path`       = '/task/executions',
  `component`  = 'TaskExecutions',
  `api_path`   = '/api/v1/task-executions*',
  `api_method` = '*',
  `type`       = 2,
  `visible`    = 1,
  `status`     = 1;

-- 子菜单排序：任务管理(1) < 执行记录(2) < 创建任务(3) < 执行详情(4) < Agent(5)
UPDATE `menus` m
INNER JOIN `menus` p ON m.`parent_id` = p.`id` AND p.`name` = 'task_dir' AND p.`deleted_at` IS NULL
SET m.`sort` = CASE m.`name`
  WHEN 'task_list' THEN 1
  WHEN 'task_executions' THEN 2
  WHEN 'task_create' THEN 3
  WHEN 'task_execution' THEN 4
  WHEN 'agent_list' THEN 5
  ELSE m.`sort`
END
WHERE m.`deleted_at` IS NULL
  AND m.`name` IN ('task_list', 'task_executions', 'task_create', 'task_execution', 'agent_list');

INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
CROSS JOIN `menus` m
WHERE r.`name` = 'admin'
  AND m.`name` = 'task_executions'
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
