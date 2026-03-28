SET @exists := (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'request_templates'
    AND COLUMN_NAME = 'notify_channels'
);

SET @sql := IF(
  @exists = 0,
  'ALTER TABLE request_templates ADD COLUMN notify_channels JSON NULL AFTER notify_applicant',
  'SELECT 1'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE request_templates
SET notify_channels = JSON_ARRAY('in_app')
WHERE notify_channels IS NULL;
