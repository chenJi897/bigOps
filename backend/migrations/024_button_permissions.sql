-- 024_button_permissions.sql
-- 按钮级权限：为每个页面插入 type=3 的按钮权限节点，幂等

-- 清理旧测试数据
DELETE FROM menus WHERE type = 3 AND name = 'ttt';

-- ==================== 系统管理 ====================
-- 用户管理(parent=2)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('user:create','新增用户','','','/api/v1/users','POST',3,2,1,1,1)
ON DUPLICATE KEY UPDATE title='新增用户',api_path='/api/v1/users',api_method='POST',type=3,parent_id=2;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('user:edit','编辑用户','','','/api/v1/users/*','POST',3,2,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑用户',api_path='/api/v1/users/*',api_method='POST',type=3,parent_id=2;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('user:delete','删除用户','','','/api/v1/users/*/delete','POST',3,2,3,1,1)
ON DUPLICATE KEY UPDATE title='删除用户',api_path='/api/v1/users/*/delete',api_method='POST',type=3,parent_id=2;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('user:assign_role','分配角色','','','/api/v1/users/*/roles','POST',3,2,4,1,1)
ON DUPLICATE KEY UPDATE title='分配角色',api_path='/api/v1/users/*/roles',api_method='POST',type=3,parent_id=2;

-- 角色管理(parent=3)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('role:create','新增角色','','','/api/v1/roles','POST',3,3,1,1,1)
ON DUPLICATE KEY UPDATE title='新增角色',api_path='/api/v1/roles',api_method='POST',type=3,parent_id=3;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('role:edit','编辑角色','','','/api/v1/roles/*','POST',3,3,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑角色',api_path='/api/v1/roles/*',api_method='POST',type=3,parent_id=3;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('role:delete','删除角色','','','/api/v1/roles/*/delete','POST',3,3,3,1,1)
ON DUPLICATE KEY UPDATE title='删除角色',api_path='/api/v1/roles/*/delete',api_method='POST',type=3,parent_id=3;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('role:assign_menu','分配菜单','','','/api/v1/roles/*/menus','POST',3,3,4,1,1)
ON DUPLICATE KEY UPDATE title='分配菜单',api_path='/api/v1/roles/*/menus',api_method='POST',type=3,parent_id=3;

-- 菜单管理(parent=4)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('menu:create','新增菜单','','','/api/v1/menus','POST',3,4,1,1,1)
ON DUPLICATE KEY UPDATE title='新增菜单',api_path='/api/v1/menus',api_method='POST',type=3,parent_id=4;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('menu:edit','编辑菜单','','','/api/v1/menus/*','POST',3,4,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑菜单',api_path='/api/v1/menus/*',api_method='POST',type=3,parent_id=4;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('menu:delete','删除菜单','','','/api/v1/menus/*/delete','POST',3,4,3,1,1)
ON DUPLICATE KEY UPDATE title='删除菜单',api_path='/api/v1/menus/*/delete',api_method='POST',type=3,parent_id=4;

-- 部门管理(parent=13)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('dept:create','新增部门','','','/api/v1/departments','POST',3,13,1,1,1)
ON DUPLICATE KEY UPDATE title='新增部门',api_path='/api/v1/departments',api_method='POST',type=3,parent_id=13;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('dept:edit','编辑部门','','','/api/v1/departments/*','POST',3,13,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑部门',api_path='/api/v1/departments/*',api_method='POST',type=3,parent_id=13;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('dept:delete','删除部门','','','/api/v1/departments/*/delete','POST',3,13,3,1,1)
ON DUPLICATE KEY UPDATE title='删除部门',api_path='/api/v1/departments/*/delete',api_method='POST',type=3,parent_id=13;

-- ==================== CMDB ====================
-- 主机资产(parent=11)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('asset:create','新增资产','','','/api/v1/assets','POST',3,11,1,1,1)
ON DUPLICATE KEY UPDATE title='新增资产',api_path='/api/v1/assets',api_method='POST',type=3,parent_id=11;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('asset:edit','编辑资产','','','/api/v1/assets/*','POST',3,11,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑资产',api_path='/api/v1/assets/*',api_method='POST',type=3,parent_id=11;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('asset:delete','删除资产','','','/api/v1/assets/*/delete','POST',3,11,3,1,1)
ON DUPLICATE KEY UPDATE title='删除资产',api_path='/api/v1/assets/*/delete',api_method='POST',type=3,parent_id=11;

-- 云账号(parent=10)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cloud_account:create','新增云账号','','','/api/v1/cloud-accounts','POST',3,10,1,1,1)
ON DUPLICATE KEY UPDATE title='新增云账号',api_path='/api/v1/cloud-accounts',api_method='POST',type=3,parent_id=10;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cloud_account:edit','编辑云账号','','','/api/v1/cloud-accounts/*','POST',3,10,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑云账号',api_path='/api/v1/cloud-accounts/*',api_method='POST',type=3,parent_id=10;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cloud_account:delete','删除云账号','','','/api/v1/cloud-accounts/*/delete','POST',3,10,3,1,1)
ON DUPLICATE KEY UPDATE title='删除云账号',api_path='/api/v1/cloud-accounts/*/delete',api_method='POST',type=3,parent_id=10;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cloud_account:sync','手动同步','','','/api/v1/cloud-accounts/*/sync','POST',3,10,4,1,1)
ON DUPLICATE KEY UPDATE title='手动同步',api_path='/api/v1/cloud-accounts/*/sync',api_method='POST',type=3,parent_id=10;

-- 服务树(parent=9)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('service_tree:create','新增节点','','','/api/v1/service-trees','POST',3,9,1,1,1)
ON DUPLICATE KEY UPDATE title='新增节点',api_path='/api/v1/service-trees',api_method='POST',type=3,parent_id=9;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('service_tree:edit','编辑节点','','','/api/v1/service-trees/*','POST',3,9,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑节点',api_path='/api/v1/service-trees/*',api_method='POST',type=3,parent_id=9;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('service_tree:delete','删除节点','','','/api/v1/service-trees/*/delete','POST',3,9,3,1,1)
ON DUPLICATE KEY UPDATE title='删除节点',api_path='/api/v1/service-trees/*/delete',api_method='POST',type=3,parent_id=9;

-- ==================== 工单系统 ====================
-- 工单模板(parent=35)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('ticket_tpl:create','新增模板','','','/api/v1/request-templates','POST',3,35,1,1,1)
ON DUPLICATE KEY UPDATE title='新增模板',api_path='/api/v1/request-templates',api_method='POST',type=3,parent_id=35;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('ticket_tpl:edit','编辑模板','','','/api/v1/request-templates/*','POST',3,35,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑模板',api_path='/api/v1/request-templates/*',api_method='POST',type=3,parent_id=35;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('ticket_tpl:delete','删除模板','','','/api/v1/request-templates/*/delete','POST',3,35,3,1,1)
ON DUPLICATE KEY UPDATE title='删除模板',api_path='/api/v1/request-templates/*/delete',api_method='POST',type=3,parent_id=35;

-- ==================== 监控中心 ====================
-- 告警规则(parent=42)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('alert_rule:create','新增规则','','','/api/v1/alert-rules','POST',3,42,1,1,1)
ON DUPLICATE KEY UPDATE title='新增规则',api_path='/api/v1/alert-rules',api_method='POST',type=3,parent_id=42;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('alert_rule:edit','编辑规则','','','/api/v1/alert-rules/*','POST',3,42,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑规则',api_path='/api/v1/alert-rules/*',api_method='POST',type=3,parent_id=42;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('alert_rule:delete','删除规则','','','/api/v1/alert-rules/*/delete','POST',3,42,3,1,1)
ON DUPLICATE KEY UPDATE title='删除规则',api_path='/api/v1/alert-rules/*/delete',api_method='POST',type=3,parent_id=42;

-- 告警静默(parent=53)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('silence:create','新增静默','','','/api/v1/alert-silences','POST',3,53,1,1,1)
ON DUPLICATE KEY UPDATE title='新增静默',api_path='/api/v1/alert-silences',api_method='POST',type=3,parent_id=53;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('silence:edit','编辑静默','','','/api/v1/alert-silences/*','POST',3,53,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑静默',api_path='/api/v1/alert-silences/*',api_method='POST',type=3,parent_id=53;

-- OnCall 值班(parent=54)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('oncall:create','新增值班表','','','/api/v1/oncall-schedules','POST',3,54,1,1,1)
ON DUPLICATE KEY UPDATE title='新增值班表',api_path='/api/v1/oncall-schedules',api_method='POST',type=3,parent_id=54;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('oncall:edit','编辑值班表','','','/api/v1/oncall-schedules/*','POST',3,54,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑值班表',api_path='/api/v1/oncall-schedules/*',api_method='POST',type=3,parent_id=54;

-- 发送组(parent=61)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('notify_group:create','新增发送组','','','/api/v1/notify-groups','POST',3,61,1,1,1)
ON DUPLICATE KEY UPDATE title='新增发送组',api_path='/api/v1/notify-groups',api_method='POST',type=3,parent_id=61;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('notify_group:edit','编辑发送组','','','/api/v1/notify-groups/*','POST',3,61,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑发送组',api_path='/api/v1/notify-groups/*',api_method='POST',type=3,parent_id=61;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('notify_group:delete','删除发送组','','','/api/v1/notify-groups/*/delete','POST',3,61,3,1,1)
ON DUPLICATE KEY UPDATE title='删除发送组',api_path='/api/v1/notify-groups/*/delete',api_method='POST',type=3,parent_id=61;

-- ==================== 任务中心 ====================
-- 任务管理(parent=20)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('task:create','新增任务','','','/api/v1/tasks','POST',3,20,1,1,1)
ON DUPLICATE KEY UPDATE title='新增任务',api_path='/api/v1/tasks',api_method='POST',type=3,parent_id=20;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('task:edit','编辑任务','','','/api/v1/tasks/*','POST',3,20,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑任务',api_path='/api/v1/tasks/*',api_method='POST',type=3,parent_id=20;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('task:delete','删除任务','','','/api/v1/tasks/*/delete','POST',3,20,3,1,1)
ON DUPLICATE KEY UPDATE title='删除任务',api_path='/api/v1/tasks/*/delete',api_method='POST',type=3,parent_id=20;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('task:execute','执行任务','','','/api/v1/tasks/*/execute','POST',3,20,4,1,1)
ON DUPLICATE KEY UPDATE title='执行任务',api_path='/api/v1/tasks/*/execute',api_method='POST',type=3,parent_id=20;

-- ==================== CI/CD ====================
-- 项目管理(parent=44)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_project:create','新增项目','','','/api/v1/cicd/projects','POST',3,44,1,1,1)
ON DUPLICATE KEY UPDATE title='新增项目',api_path='/api/v1/cicd/projects',api_method='POST',type=3,parent_id=44;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_project:edit','编辑项目','','','/api/v1/cicd/projects/*','POST',3,44,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑项目',api_path='/api/v1/cicd/projects/*',api_method='POST',type=3,parent_id=44;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_project:delete','删除项目','','','/api/v1/cicd/projects/*/delete','POST',3,44,3,1,1)
ON DUPLICATE KEY UPDATE title='删除项目',api_path='/api/v1/cicd/projects/*/delete',api_method='POST',type=3,parent_id=44;

-- 流水线管理(parent=45)
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_pipeline:create','新增流水线','','','/api/v1/cicd/pipelines','POST',3,45,1,1,1)
ON DUPLICATE KEY UPDATE title='新增流水线',api_path='/api/v1/cicd/pipelines',api_method='POST',type=3,parent_id=45;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_pipeline:edit','编辑流水线','','','/api/v1/cicd/pipelines/*','POST',3,45,2,1,1)
ON DUPLICATE KEY UPDATE title='编辑流水线',api_path='/api/v1/cicd/pipelines/*',api_method='POST',type=3,parent_id=45;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_pipeline:delete','删除流水线','','','/api/v1/cicd/pipelines/*/delete','POST',3,45,3,1,1)
ON DUPLICATE KEY UPDATE title='删除流水线',api_path='/api/v1/cicd/pipelines/*/delete',api_method='POST',type=3,parent_id=45;
INSERT INTO menus (name, title, path, component, api_path, api_method, type, parent_id, sort, visible, status)
VALUES ('cicd_pipeline:run','手动运行','','','/api/v1/cicd/pipelines/*/run','POST',3,45,4,1,1)
ON DUPLICATE KEY UPDATE title='手动运行',api_path='/api/v1/cicd/pipelines/*/run',api_method='POST',type=3,parent_id=45;

-- ==================== 角色分配按钮权限 ====================
-- admin(role_id=1) 获得所有按钮权限
INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT 1, id FROM menus WHERE type = 3;

-- ops(role_id=4) 获得所有按钮权限（运维需要全部操作权限）
INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT 4, id FROM menus WHERE type = 3;
