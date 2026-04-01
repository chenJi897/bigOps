-- 019_notification_templates.sql
-- Seed default notification templates + add notify_template to alert_rules/cicd_pipelines

-- notification_templates 表由 GORM AutoMigrate 创建，这里只 seed 默认模板数据

INSERT INTO `notification_templates` (`event_type`, `title`, `content`, `variables`, `is_default`) VALUES
('alert_firing', '告警触发：{{.rule_name}}', '## 告警触发：{{.rule_name}}\n\n| 项目 | 值 |\n|---|---|\n| 主机 | {{.hostname}}({{.ip}}) |\n| 指标 | {{.metric_type}} |\n| 当前值 | {{.metric_value}} |\n| 阈值 | {{.threshold}} |\n\n> 请及时处理', 'rule_name, hostname, ip, metric_type, metric_value, threshold', 1),
('alert_resolved', '告警恢复：{{.rule_name}}', '## 告警恢复：{{.rule_name}}\n\n| 项目 | 值 |\n|---|---|\n| 主机 | {{.hostname}}({{.ip}}) |\n| 指标 | {{.metric_type}} |\n| 当前值 | {{.metric_value}} |\n\n> 告警已恢复', 'rule_name, hostname, ip, metric_type, metric_value', 1),
('pipeline_succeeded', '流水线成功：{{.pipeline_name}}', '## 流水线运行成功\n\n| 项目 | 值 |\n|---|---|\n| 流水线 | {{.pipeline_name}} |\n| 分支 | {{.branch}} |\n| 结果 | {{.result}} |\n| 状态 | {{.status}} |', 'pipeline_name, branch, result, status, pipeline_id, run_id', 1),
('pipeline_failed', '流水线失败：{{.pipeline_name}}', '## 流水线运行失败\n\n| 项目 | 值 |\n|---|---|\n| 流水线 | {{.pipeline_name}} |\n| 分支 | {{.branch}} |\n| 结果 | {{.result}} |\n| 状态 | {{.status}} |\n\n> 请检查失败原因', 'pipeline_name, branch, result, status, pipeline_id, run_id', 1),
('approval_pending', '审批待处理：{{.ticket_no}}', '## 审批待处理\n\n| 项目 | 值 |\n|---|---|\n| 工单号 | {{.ticket_no}} |\n| 工单标题 | {{.ticket_title}} |\n| 审批阶段 | {{.stage_name}} |\n\n> 请尽快审批', 'ticket_id, ticket_no, ticket_title, stage_no, stage_name', 1),
('approval_approved', '审批已通过：{{.ticket_no}}', '## 审批已通过\n\n| 项目 | 值 |\n|---|---|\n| 工单号 | {{.ticket_no}} |\n\n> 审批流程已完成', 'ticket_id, ticket_no, instance_id', 1),
('approval_rejected', '审批已拒绝：{{.ticket_no}}', '## 审批已拒绝\n\n| 项目 | 值 |\n|---|---|\n| 工单号 | {{.ticket_no}} |\n| 审批阶段 | {{.stage_name}} |\n| 审批意见 | {{.comment}} |\n\n> 请根据审批意见修改后重新提交', 'ticket_id, ticket_no, stage_no, stage_name, approver_id, action, comment', 1),
('notification_test', '通知测试', '## 通知渠道测试\n\n这是一条测试消息，用于验证通知通道是否正常工作。\n\n- 标题：{{.title}}\n- 内容：{{.content}}', 'title, content, channels', 1)
ON DUPLICATE KEY UPDATE `title` = VALUES(`title`), `content` = VALUES(`content`), `variables` = VALUES(`variables`);

-- alert_rules 增加 notify_template 字段
ALTER TABLE `alert_rules` ADD COLUMN IF NOT EXISTS `notify_template` TEXT DEFAULT NULL;

-- cicd_pipelines 增加 notify_template 字段
ALTER TABLE `cicd_pipelines` ADD COLUMN IF NOT EXISTS `notify_template` TEXT DEFAULT NULL;
