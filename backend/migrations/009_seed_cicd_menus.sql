-- 009_seed_cicd_menus.sql
-- CI/CD 模块菜单：项目管理 / 流水线管理

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `type`, `sort`, `visible`, `status`)
VALUES (0, 'cicd_dir', 'CI/CD', 'Promotion', '/cicd', '', 1, 110, 1, 1)
ON DUPLICATE KEY UPDATE
  `title` = 'CI/CD',
  `icon` = 'Promotion',
  `path` = '/cicd',
  `component` = '',
  `type` = 1,
  `sort` = 110,
  `visible` = 1,
  `status` = 1;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'cicd_projects', '项目管理', 'FolderOpened', '/cicd/projects', 'CicdProjects', '/api/v1/cicd/projects*', '*', 2, 1, 1, 1
FROM `menus` m WHERE m.`name` = 'cicd_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = '项目管理',
  `icon` = 'FolderOpened',
  `path` = '/cicd/projects',
  `component` = 'CicdProjects',
  `api_path` = '/api/v1/cicd/projects*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 1,
  `visible` = 1,
  `status` = 1;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'cicd_pipelines', '流水线管理', 'Switch', '/cicd/pipelines', 'CicdPipelines', '/api/v1/cicd/pipelines*', '*', 2, 2, 1, 1
FROM `menus` m WHERE m.`name` = 'cicd_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = '流水线管理',
  `icon` = 'Switch',
  `path` = '/cicd/pipelines',
  `component` = 'CicdPipelines',
  `api_path` = '/api/v1/cicd/pipelines*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 2,
  `visible` = 1,
  `status` = 1;

INSERT INTO `menus` (`parent_id`, `name`, `title`, `icon`, `path`, `component`, `api_path`, `api_method`, `type`, `sort`, `visible`, `status`)
SELECT m.id, 'cicd_runs', '运行记录', 'List', '/cicd/runs', 'CicdRuns', '/api/v1/cicd/runs*', '*', 2, 3, 1, 1
FROM `menus` m WHERE m.`name` = 'cicd_dir' AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE
  `parent_id` = VALUES(`parent_id`),
  `title` = '运行记录',
  `icon` = 'List',
  `path` = '/cicd/runs',
  `component` = 'CicdRuns',
  `api_path` = '/api/v1/cicd/runs*',
  `api_method` = '*',
  `type` = 2,
  `sort` = 3,
  `visible` = 1,
  `status` = 1;

INSERT INTO `role_menus` (`role_id`, `menu_id`)
SELECT r.id, m.id
FROM `roles` r
JOIN `menus` m ON m.`name` IN ('cicd_dir', 'cicd_projects', 'cicd_pipelines', 'cicd_runs')
WHERE r.`name` IN ('admin', 'ops')
  AND r.`deleted_at` IS NULL
  AND m.`deleted_at` IS NULL
ON DUPLICATE KEY UPDATE `role_id` = `role_id`;
