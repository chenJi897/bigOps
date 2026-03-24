-- 004_seed_ticket_approval_menus.sql
-- 为管理员默认补充工单中心、审批待办、请求模板、审批策略、通知联调菜单

-- 工单中心目录
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'ticket_dir', '工单中心', 'Tickets', '', '', 1, 110, 1, 1)
ON DUPLICATE KEY UPDATE `title` = '工单中心', `icon` = 'Tickets', `path` = '', `component` = '';

-- 工单列表
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_list', '工单列表', 'Tickets', '/tickets', 'TicketList', '/api/v1/tickets', 'GET', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir'
ON DUPLICATE KEY UPDATE `title` = '工单列表', `icon` = 'Tickets', `path` = '/tickets', `component` = 'TicketList';

-- 审批待办
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'approval_inbox', '审批待办', 'Bell', '/approval/inbox', 'ApprovalInbox', '/api/v1/approval-instances/pending', 'GET', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir'
ON DUPLICATE KEY UPDATE `title` = '审批待办', `icon` = 'Bell', `path` = '/approval/inbox', `component` = 'ApprovalInbox';

-- 工单类型管理
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'ticket_types', '工单类型', 'List', '/ticket/types', 'TicketTypes', '/api/v1/ticket-types', 'GET', 2, 3, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir'
ON DUPLICATE KEY UPDATE `title` = '工单类型', `icon` = 'List', `path` = '/ticket/types', `component` = 'TicketTypes';

-- 请求模板
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'request_templates', '请求模板', 'DocumentAdd', '/request/templates', 'RequestTemplates', '/api/v1/request-templates', 'GET', 2, 4, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir'
ON DUPLICATE KEY UPDATE `title` = '请求模板', `icon` = 'DocumentAdd', `path` = '/request/templates', `component` = 'RequestTemplates';

-- 审批策略
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'approval_policies', '审批策略', 'Connection', '/approval/policies', 'ApprovalPolicies', '/api/v1/approval-policies', 'GET', 2, 5, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir'
ON DUPLICATE KEY UPDATE `title` = '审批策略', `icon` = 'Connection', `path` = '/approval/policies', `component` = 'ApprovalPolicies';

-- 通知联调
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'notification_console', '通知联调', 'Message', '/notification/console', 'NotificationConsole', '/api/v1/notifications/events', 'GET', 2, 6, 1, 1
FROM `menus` m WHERE m.`name` = 'ticket_dir'
ON DUPLICATE KEY UPDATE `title` = '通知联调', `icon` = 'Message', `path` = '/notification/console', `component` = 'NotificationConsole';

-- 默认给 admin 角色分配工单中心相关菜单
INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
JOIN `menus` m ON m.`name` IN (
  'ticket_dir',
  'ticket_list',
  'approval_inbox',
  'ticket_types',
  'request_templates',
  'approval_policies',
  'notification_console'
)
WHERE r.`name` = 'admin'
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
