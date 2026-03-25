-- 006_fix_all_menu_api_paths.sql
-- 所有菜单 api_path 改为通配前缀模式，api_method 改为 *
-- Casbin matcher 使用 keyMatch2 + act=* 通配
-- 一个菜单覆盖该模块所有子路径和 HTTP 方法

-- 系统管理
UPDATE `menus` SET `api_path` = '/api/v1/users*',        `api_method` = '*' WHERE `name` = 'user_list'              AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/roles*',        `api_method` = '*' WHERE `name` = 'role_list'              AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/menus*',        `api_method` = '*' WHERE `name` = 'menu_list'              AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/audit-logs*',   `api_method` = '*' WHERE `name` = 'audit_logs'             AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/departments*',  `api_method` = '*' WHERE `name` = 'department_management'  AND `deleted_at` IS NULL;

-- CMDB
UPDATE `menus` SET `api_path` = '/api/v1/service-trees*',  `api_method` = '*' WHERE `name` = 'service_tree'   AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/cloud-accounts*', `api_method` = '*' WHERE `name` = 'cloud_accounts' AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/assets*',         `api_method` = '*' WHERE `name` = 'assets'         AND `deleted_at` IS NULL;

-- 工单
UPDATE `menus` SET `api_path` = '/api/v1/tickets*',      `api_method` = '*' WHERE `name` = 'ticket_list'    AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/tickets*',      `api_method` = '*' WHERE `name` = 'ticket_create'  AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/tickets*',      `api_method` = '*' WHERE `name` = 'ticket_detail'  AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/ticket-types*', `api_method` = '*' WHERE `name` = 'ticket_types'   AND `deleted_at` IS NULL;

-- 任务中心
UPDATE `menus` SET `api_path` = '/api/v1/tasks*',             `api_method` = '*' WHERE `name` = 'task_list'      AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/tasks*',             `api_method` = '*' WHERE `name` = 'task_create'    AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/task-executions*',   `api_method` = '*' WHERE `name` = 'task_execution' AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/agents*',            `api_method` = '*' WHERE `name` = 'agent_list'     AND `deleted_at` IS NULL;

-- 重建 Casbin policy
DELETE FROM `casbin_rule` WHERE `ptype` = 'p';

INSERT INTO `casbin_rule` (`ptype`, `v0`, `v1`, `v2`)
SELECT 'p', r.`name`, m.`api_path`, m.`api_method`
FROM `roles` r
JOIN `role_menus` rm ON rm.`role_id` = r.`id`
JOIN `menus` m ON m.`id` = rm.`menu_id`
WHERE r.`name` != 'admin'
  AND r.`deleted_at` IS NULL
  AND m.`deleted_at` IS NULL
  AND m.`api_path` IS NOT NULL AND m.`api_path` != ''
  AND m.`api_method` IS NOT NULL AND m.`api_method` != ''
ON DUPLICATE KEY UPDATE `v0` = VALUES(`v0`);
