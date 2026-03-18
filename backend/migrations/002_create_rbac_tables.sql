-- 角色表
CREATE TABLE IF NOT EXISTS `roles` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '角色ID',
  `name` VARCHAR(50) NOT NULL COMMENT '角色标识（英文，如 admin, viewer）',
  `display_name` VARCHAR(100) NOT NULL COMMENT '角色显示名（如 系统管理员）',
  `description` VARCHAR(255) DEFAULT NULL COMMENT '角色描述',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=启用 0=禁用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色表';

-- 菜单/权限表（树形结构，既表示前端菜单也表示 API 权限）
CREATE TABLE IF NOT EXISTS `menus` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '菜单ID',
  `parent_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父菜单ID，0 表示顶级',
  `name` VARCHAR(50) NOT NULL COMMENT '菜单标识（英文，如 user_list）',
  `title` VARCHAR(100) NOT NULL COMMENT '菜单显示名（如 用户列表）',
  `icon` VARCHAR(100) DEFAULT NULL COMMENT '菜单图标',
  `path` VARCHAR(255) DEFAULT NULL COMMENT '前端路由路径',
  `component` VARCHAR(255) DEFAULT NULL COMMENT '前端组件路径',
  `api_path` VARCHAR(255) DEFAULT NULL COMMENT 'API 路径（如 /api/v1/users）',
  `api_method` VARCHAR(10) DEFAULT NULL COMMENT 'HTTP 方法（GET, POST, PUT, DELETE）',
  `type` TINYINT NOT NULL DEFAULT 1 COMMENT '类型：1=目录 2=菜单 3=按钮/API权限',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序值',
  `visible` TINYINT NOT NULL DEFAULT 1 COMMENT '是否在菜单中可见：1=是 0=否',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态：1=启用 0=禁用',
  `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` DATETIME DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`),
  KEY `idx_parent_id` (`parent_id`),
  KEY `idx_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='菜单/权限表';

-- 用户-角色关联表
CREATE TABLE IF NOT EXISTS `user_roles` (
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  PRIMARY KEY (`user_id`, `role_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户角色关联表';

-- 角色-菜单关联表
CREATE TABLE IF NOT EXISTS `role_menus` (
  `role_id` BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
  `menu_id` BIGINT UNSIGNED NOT NULL COMMENT '菜单ID',
  PRIMARY KEY (`role_id`, `menu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='角色菜单关联表';

-- 初始化管理员角色
INSERT INTO `roles` (`name`, `display_name`, `description`, `sort`, `status`)
VALUES ('admin', '系统管理员', '拥有全部权限', 1, 1)
ON DUPLICATE KEY UPDATE `name` = `name`;

-- 给 admin 用户分配管理员角色
INSERT INTO `user_roles` (`user_id`, `role_id`)
SELECT 1, id FROM `roles` WHERE `name` = 'admin'
ON DUPLICATE KEY UPDATE `user_id` = `user_id`;

-- 初始化默认菜单
INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `type`, `sort`, `visible`) VALUES
(0, 'system',       '系统管理', 'Setting',  '/system',       1, 100, 1),
(1, 'user_list',    '用户管理', 'User',     '/system/users', 2, 1,   1),
(1, 'role_list',    '角色管理', 'Key',      '/system/roles', 2, 2,   1),
(1, 'menu_list',    '菜单管理', 'Menu',     '/system/menus', 2, 3,   1)
ON DUPLICATE KEY UPDATE `name` = `name`;

-- 给管理员角色分配全部菜单
INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id FROM `roles` r, `menus` m WHERE r.`name` = 'admin'
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
