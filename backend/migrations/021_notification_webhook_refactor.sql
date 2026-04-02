-- 021_notification_webhook_refactor.sql
-- 通知系统重构：去 Message Pusher，Webhook 内嵌到业务配置

-- 1. alert_rules 增加 notify_config（渠道→webhook 配置 JSON）
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS notify_config TEXT COMMENT '渠道webhook配置JSON' AFTER notify_channels;

-- 2. request_templates 增加 notify_config
ALTER TABLE request_templates ADD COLUMN IF NOT EXISTS notify_config TEXT COMMENT '渠道webhook配置JSON' AFTER notify_channels;

-- 3. notification_deliveries 增加 webhook 投递信息
ALTER TABLE notification_deliveries ADD COLUMN IF NOT EXISTS webhook_url VARCHAR(500) DEFAULT '' AFTER recipient;
ALTER TABLE notification_deliveries ADD COLUMN IF NOT EXISTS webhook_secret VARCHAR(200) DEFAULT '' AFTER webhook_url;
