SET @exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'alert_rules'
    AND COLUMN_NAME = 'notify_channels'
);

SET @sql := IF(
  @exists = 0,
  'ALTER TABLE alert_rules ADD COLUMN notify_channels JSON NULL AFTER notify_user_ids',
  'SELECT 1'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE alert_rules
SET notify_channels = JSON_ARRAY('in_app')
WHERE notify_channels IS NULL;
