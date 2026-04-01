SET @exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'notification_user_settings'
    AND COLUMN_NAME = 'channel_targets'
);

SET @sql := IF(
  @exists = 0,
  'ALTER TABLE notification_user_settings ADD COLUMN channel_targets JSON NULL AFTER subscribed_biz_types',
  'SELECT 1'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE notification_user_settings
SET channel_targets = JSON_OBJECT()
WHERE channel_targets IS NULL;

