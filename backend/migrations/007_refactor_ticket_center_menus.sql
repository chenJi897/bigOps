-- 007_refactor_ticket_center_menus.sql
-- 将工单管理导航收敛为发起工单 / 我的待办 / 我的申请 / 工单模板，并隐藏旧入口

-- 1. 父目录统一为“工单管理”，保持工单主题图标与基础路径
UPDATE `menus`
SET `title` = '工单管理',
    `icon` = 'Tickets',
    `path` = '/ticket',
    `component` = '',
    `type` = 1,
    `sort` = 110,
    `visible` = 1,
    `status` = 1
WHERE `name` = 'ticket_dir' AND `deleted_at` IS NULL;

-- 2. 复用原 ticket_create 菜单，重命名为 ticket_launch 并覆盖文案/路径/API
UPDATE `menus`
SET `name` = 'ticket_launch',
    `title` = '发起工单',
    `icon` = 'Document',
    `path` = '/ticket/create',
    `component` = 'TicketCreate',
    `api_path` = '/api/v1/tickets*',
    `api_method` = '*',
    `type` = 2,
    `sort` = 1,
    `visible` = 1,
    `status` = 1
WHERE `name` = 'ticket_create' AND `deleted_at` IS NULL;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_launch', '发起工单', 'Document', '/ticket/create', 'TicketCreate', '/api/v1/tickets*', '*', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '发起工单',
  `icon`       = 'Document',
  `path`       = '/ticket/create',
  `component`  = 'TicketCreate',
  `api_path`   = '/api/v1/tickets*',
  `api_method` = '*',
  `type`       = 2,
  `sort`       = 1,
  `visible`    = 1,
  `status`     = 1;

-- 3. 新增“我的待办”和“我的申请”入口
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_todo', '我的待办', 'Bell', '/ticket/todo', 'TicketList', '/api/v1/tickets*', '*', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `title` = '我的待办',
  `icon` = 'Bell',
  `path` = '/ticket/todo',
  `component` = 'TicketList',
  `api_path` = '/api/v1/tickets*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 2,
  `visible` = 1,
  `status` = 1;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_applied', '我的申请', 'Document', '/ticket/applied', 'TicketList', '/api/v1/tickets*', '*', 2, 3, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `title` = '我的申请',
  `icon` = 'Document',
  `path` = '/ticket/applied',
  `component` = 'TicketList',
  `api_path` = '/api/v1/tickets*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 3,
  `visible` = 1,
  `status` = 1;

-- 4. 将旧的请求模板菜单重命名为 ticket_templates，并作为工单模板入口
UPDATE `menus`
SET `name` = 'ticket_templates',
    `title` = '工单模板',
    `icon` = 'DocumentAdd',
    `path` = '/ticket/templates',
    `component` = 'RequestTemplates',
    `api_path` = '/api/v1/request-templates*',
    `api_method` = '*',
    `type` = 2,
    `sort` = 4,
    `visible` = 1,
    `status` = 1
WHERE `name` = 'request_templates' AND `deleted_at` IS NULL;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_templates', '工单模板', 'DocumentAdd', '/ticket/templates', 'RequestTemplates', '/api/v1/request-templates*', '*', 2, 4, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id`  = VALUES(`parent_id`),
  `title`      = '工单模板',
  `icon`       = 'DocumentAdd',
  `path`       = '/ticket/templates',
  `component`  = 'RequestTemplates',
  `api_path`   = '/api/v1/request-templates*',
  `api_method` = '*',
  `type`       = 2,
  `sort`       = 4,
  `visible`    = 1,
  `status`     = 1;

-- 5. 隐藏旧的重复入口
UPDATE `menus`
SET `visible` = 0
WHERE `name` IN ('ticket_list', 'approval_inbox', 'ticket_types', 'approval_policies', 'ticket_all', 'ticket_request', 'ticket_mine')
  AND `deleted_at` IS NULL;

-- 6. 迁移角色权限，确保原有工单相关角色仍拥有新的入口
INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT DISTINCT rm.role_id, nm.id
FROM `role_menus` rm
JOIN `menus` om ON om.id = rm.menu_id
JOIN `menus` nm ON nm.name IN ('ticket_launch', 'ticket_todo', 'ticket_applied', 'ticket_templates')
WHERE om.`name` IN ('ticket_list', 'ticket_launch', 'ticket_detail', 'approval_inbox', 'ticket_types', 'ticket_templates', 'approval_policies')
  AND om.`deleted_at` IS NULL
  AND nm.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `role_id` = VALUES(`role_id`),
  `menu_id` = VALUES(`menu_id`);
