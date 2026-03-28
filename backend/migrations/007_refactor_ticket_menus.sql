-- 007_refactor_ticket_menus.sql
-- 工单中心菜单重构：单列表 → 多入口独立视图
-- 参照爱猫家工单系统：发起工单/我的待办/我的申请/所有工单/工单模板

-- ============================================================
-- 1. 更新目录节点
-- ============================================================
UPDATE `menus` SET `title` = '工单中心', `icon` = 'Tickets', `path` = '/ticket'
  WHERE `name` = 'ticket_dir' AND `deleted_at` IS NULL;

-- ============================================================
-- 2. ticket_list → ticket_all（所有工单）
-- ============================================================
UPDATE `menus` SET
  `name` = 'ticket_all', `title` = '所有工单', `icon` = 'Folder',
  `path` = '/ticket/all', `component` = 'TicketList',
  `api_path` = '/api/v1/tickets*', `api_method` = '*',
  `sort` = 40, `visible` = 1
WHERE `name` IN ('ticket_list', 'ticket_all') AND `deleted_at` IS NULL;

-- ============================================================
-- 3. 新增 ticket_request（发起工单）
-- ============================================================
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_request', '发起工单', 'Edit', '/ticket/request', 'TicketRequest', '/api/v1/tickets*', '*', 2, 10, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '发起工单',
  `icon`       = 'Edit',
  `path`       = '/ticket/request',
  `component`  = 'TicketRequest',
  `api_path`   = '/api/v1/tickets*',
  `api_method` = '*',
  `sort`       = 10,
  `visible`    = 1,
  `status`     = 1;

-- ============================================================
-- 4. 新增 ticket_todo（我的待办）
-- ============================================================
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_todo', '我的待办', 'Bell', '/ticket/todo', 'TicketTodo', '/api/v1/tickets*', '*', 2, 20, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '我的待办',
  `icon`       = 'Bell',
  `path`       = '/ticket/todo',
  `component`  = 'TicketTodo',
  `api_path`   = '/api/v1/tickets*',
  `api_method` = '*',
  `sort`       = 20,
  `visible`    = 1,
  `status`     = 1;

-- ============================================================
-- 5. 新增 ticket_mine（我的申请）
-- ============================================================
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_mine', '我的申请', 'User', '/ticket/mine', 'TicketMine', '/api/v1/tickets*', '*', 2, 30, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '我的申请',
  `icon`       = 'User',
  `path`       = '/ticket/mine',
  `component`  = 'TicketMine',
  `api_path`   = '/api/v1/tickets*',
  `api_method` = '*',
  `sort`       = 30,
  `visible`    = 1,
  `status`     = 1;

-- ============================================================
-- 6. ticket_types → 工单模板（改名 + 调整排序）
-- ============================================================
UPDATE `menus` SET
  `title` = '工单模板', `icon` = 'Document',
  `sort` = 50
WHERE `name` = 'ticket_types' AND `deleted_at` IS NULL;

-- ============================================================
-- 7. 隐藏页面调整
-- ============================================================
-- ticket_create 保持隐藏
UPDATE `menus` SET `visible` = 0 WHERE `name` = 'ticket_create' AND `deleted_at` IS NULL;
-- ticket_detail 保持隐藏
UPDATE `menus` SET `visible` = 0 WHERE `name` = 'ticket_detail' AND `deleted_at` IS NULL;
-- approval_inbox 合并到我的待办，改为隐藏
UPDATE `menus` SET `visible` = 0 WHERE `name` = 'approval_inbox' AND `deleted_at` IS NULL;

-- ============================================================
-- 8. 为 ops 角色分配新菜单（如果 ops 有旧的 ticket_list，也给新菜单）
-- ============================================================
INSERT IGNORE INTO `role_menus` (`role_id`, `menu_id`)
SELECT rm.role_id, m.id
FROM `role_menus` rm
JOIN `menus` old_m ON old_m.id = rm.menu_id AND old_m.`name` = 'ticket_all'
CROSS JOIN `menus` m
WHERE m.`name` IN ('ticket_request', 'ticket_todo', 'ticket_mine')
  AND m.`deleted_at` IS NULL;

-- ============================================================
-- 9. 重建 Casbin policy
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
