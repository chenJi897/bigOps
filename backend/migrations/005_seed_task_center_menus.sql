-- 005_seed_task_center_menus.sql
-- 任务执行中心菜单 (Module 04)
-- 幂等：INSERT ... ON DUPLICATE KEY UPDATE 确保新环境插入、旧环境修正脏数据
-- 执行方式：手动 mysql < 005_seed_task_center_menus.sql，不在服务启动时自动执行

-- ============================================================
-- 1. 目录节点 task_dir
-- ============================================================
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'task_dir', '任务中心', 'Operation', '/task', '', 1, 60, 1, 1)
ON DUPLICATE KEY UPDATE
  `title`   = '任务中心',
  `icon`    = 'Operation',
  `path`    = '/task',
  `type`    = 1,
  `sort`    = 60,
  `visible` = 1,
  `status`  = 1;

-- ============================================================
-- 2. 子菜单（parent_id 通过子查询动态获取 task_dir 的 ID）
-- ============================================================

-- 任务管理（可见）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_list', '任务管理', 'List', '/task/list', 'TaskList', '/api/v1/tasks*', '*', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'task_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '任务管理',
  `icon`       = 'List',
  `path`       = '/task/list',
  `component`  = 'TaskList',
  `api_path`   = '/api/v1/tasks*',
  `api_method` = '*',
  `type`       = 2,
  `sort`       = 1,
  `visible`    = 1,
  `status`     = 1;

-- 创建任务（隐藏菜单 visible=0）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_create', '创建任务', '', '/task/create', 'TaskCreate', '/api/v1/tasks*', '*', 2, 2, 0, 1
FROM `menus` m WHERE m.`name` = 'task_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '创建任务',
  `icon`       = '',
  `path`       = '/task/create',
  `component`  = 'TaskCreate',
  `api_path`   = '/api/v1/tasks*',
  `api_method` = '*',
  `type`       = 2,
  `sort`       = 2,
  `visible`    = 0,
  `status`     = 1;

-- 执行详情（隐藏菜单 visible=0）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'task_execution', '执行详情', '', '/task/execution', 'TaskExecution', '/api/v1/task-executions*', '*', 2, 3, 0, 1
FROM `menus` m WHERE m.`name` = 'task_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '执行详情',
  `icon`       = '',
  `path`       = '/task/execution',
  `component`  = 'TaskExecution',
  `api_path`   = '/api/v1/task-executions*',
  `api_method` = '*',
  `type`       = 2,
  `sort`       = 3,
  `visible`    = 0,
  `status`     = 1;

-- Agent 管理（可见）
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'agent_list', 'Agent 管理', 'Monitor', '/task/agents', 'AgentList', '/api/v1/agents*', '*', 2, 4, 1, 1
FROM `menus` m WHERE m.`name` = 'task_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = 'Agent 管理',
  `icon`       = 'Monitor',
  `path`       = '/task/agents',
  `component`  = 'AgentList',
  `api_path`   = '/api/v1/agents*',
  `api_method` = '*',
  `type`       = 2,
  `sort`       = 4,
  `visible`    = 1,
  `status`     = 1;
