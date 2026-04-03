-- 025_cleanup_duplicate_menus.sql
-- 清理工单管理下的重复菜单项

-- 停用旧的重复菜单（保留新版）
UPDATE menus SET status = 0, visible = 0 WHERE name = 'ticket_request';   -- 旧"发起工单"，保留 ticket_launch
UPDATE menus SET status = 0, visible = 0 WHERE name = 'ticket_mine';      -- 旧"我的申请"，保留 ticket_applied
UPDATE menus SET status = 0, visible = 0 WHERE name = 'ticket_types';     -- 旧"工单模板"，保留 ticket_templates

-- 工单详情和所有工单是动态路由/管理入口，不在侧栏显示
UPDATE menus SET visible = 0 WHERE name = 'ticket_detail';
UPDATE menus SET visible = 0 WHERE name = 'ticket_all';
