-- 022_create_notify_groups.sql
-- 发送组功能：通知对象打包管理

CREATE TABLE IF NOT EXISTS notify_groups (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500) DEFAULT '',
    webhooks_json TEXT COMMENT '[{channel_type,label,webhook_url,secret}]',
    notify_user_ids JSON COMMENT '[userID1, userID2]',
    repeat_enabled TINYINT DEFAULT 0,
    repeat_interval_seconds INT DEFAULT 300,
    send_resolved TINYINT DEFAULT 1,
    escalation_enabled TINYINT DEFAULT 0,
    escalation_minutes INT DEFAULT 20,
    escalation_user_ids JSON COMMENT '[userID]',
    escalation_webhooks_json TEXT,
    status TINYINT DEFAULT 1,
    created_by BIGINT DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME DEFAULT NULL,
    UNIQUE KEY uk_name (name),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- alert_rules 增加 notify_group_id
ALTER TABLE alert_rules ADD COLUMN IF NOT EXISTS notify_group_id BIGINT DEFAULT 0 AFTER notify_config;

-- alert_events 增加重复/升级追踪字段
ALTER TABLE alert_events ADD COLUMN IF NOT EXISTS last_notify_at DATETIME DEFAULT NULL AFTER resolved_at;
ALTER TABLE alert_events ADD COLUMN IF NOT EXISTS escalated TINYINT DEFAULT 0 AFTER last_notify_at;
