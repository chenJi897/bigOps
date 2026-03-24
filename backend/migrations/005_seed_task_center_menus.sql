-- 005_seed_task_center_menus.sql
-- 任务执行中心菜单（Module 04）
-- 注意：main.go 中的 seedTaskMenus() 已实现幂等自动插入，
-- 此文件仅作为参考文档。如果通过手动 SQL 初始化，可执行此文件。

-- 任务中心目录
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'task_dir', '任务中心', 'Operation', '', '', 1, 60, 1, 1)
ON DUPLICATE KEY UPDATE `title` = '任务中心', `icon` = 'Operation';

-- 任务管理
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_list', '任务管理', 'List', '/task/list', 'TaskList', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'task_dir'
ON DUPLICATE KEY UPDATE `title` = '任务管理', `path` = '/task/list', `component` = 'TaskList';

-- 创建任务（隐藏菜单）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_create', '创建任务', '', '/task/create', 'TaskCreate', 2, 2, 0, 1
FROM `menus` m WHERE m.`name` = 'task_dir'
ON DUPLICATE KEY UPDATE `title` = '创建任务', `path` = '/task/create', `component` = 'TaskCreate';

-- 执行详情（隐藏菜单）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_execution', '执行详情', '', '/task/execution', 'TaskExecution', 2, 3, 0, 1
FROM `menus` m WHERE m.`name` = 'task_dir'
ON DUPLICATE KEY UPDATE `title` = '执行详情', `path` = '/task/execution', `component` = 'TaskExecution';

-- Agent 管理
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'agent_list', 'Agent 管理', 'Monitor', '/task/agents', 'AgentList', 2, 4, 1, 1
FROM `menus` m WHERE m.`name` = 'task_dir'
ON DUPLICATE KEY UPDATE `title` = 'Agent 管理', `path` = '/task/agents', `component` = 'AgentList';

-- 给非 admin 角色分配任务中心菜单（admin 自动拥有所有菜单）
-- 如需为其他角色分配，执行：
-- INSERT INTO `role_menus` (`role_id`, `menu_id`)
-- SELECT r.id, m.id
-- FROM `roles` r
-- CROSS JOIN `menus` m
-- WHERE r.`name` = 'your_role_name'
--   AND m.`name` IN ('task_dir', 'task_list', 'task_create', 'task_execution', 'agent_list')
--   AND m.`deleted_at` IS NULL
-- ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
