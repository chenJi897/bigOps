CREATE TABLE IF NOT EXISTS `monitor_datasources` (
  `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(128) NOT NULL UNIQUE,
  `type` VARCHAR(32) NOT NULL,
  `base_url` VARCHAR(512) NOT NULL,
  `access_type` VARCHAR(32) DEFAULT 'proxy',
  `auth_type` VARCHAR(32),
  `username` VARCHAR(128),
  `password` VARCHAR(256),
  `headers_json` JSON DEFAULT ('{}'),
  `status` VARCHAR(32) DEFAULT 'active',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME NULL,
  INDEX `idx_monitor_datasource_type` (`type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
