-- 006_fix_all_menu_api_paths.sql
-- 补全所有菜单的 api_path + api_method
-- 启用 Casbin 后，只有 api_path + api_method 非空的菜单才会生成 policy
-- 如果缺失，即使角色分配了菜单，Casbin 也不会放行对应 API

-- ============================================================
-- 系统管理模块
-- ============================================================
UPDATE `menus` SET `api_path` = '/api/v1/users',       `api_method` = 'GET' WHERE `name` = 'user_list'              AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/roles',       `api_method` = 'GET' WHERE `name` = 'role_list'              AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/menus',       `api_method` = 'GET' WHERE `name` = 'menu_list'              AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/audit-logs',  `api_method` = 'GET' WHERE `name` = 'audit_logs'             AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/departments', `api_method` = 'GET' WHERE `name` = 'department_management'  AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;

-- ============================================================
-- CMDB 模块
-- ============================================================
UPDATE `menus` SET `api_path` = '/api/v1/service-trees',  `api_method` = 'GET' WHERE `name` = 'service_tree'   AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/cloud-accounts', `api_method` = 'GET' WHERE `name` = 'cloud_accounts' AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/assets',         `api_method` = 'GET' WHERE `name` = 'assets'         AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;

-- ============================================================
-- 工单模块（补缺失的子菜单）
-- ============================================================
UPDATE `menus` SET `api_path` = '/api/v1/tickets',     `api_method` = 'POST' WHERE `name` = 'ticket_create'  AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/tickets/:id', `api_method` = 'GET'  WHERE `name` = 'ticket_detail'  AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;
UPDATE `menus` SET `api_path` = '/api/v1/ticket-types',`api_method` = 'GET'  WHERE `name` = 'ticket_types'   AND (`api_path` IS NULL OR `api_path` = '') AND `deleted_at` IS NULL;

-- ============================================================
-- 重建 Casbin policy（清除旧的，根据角色-菜单关系重建）
-- ============================================================
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
