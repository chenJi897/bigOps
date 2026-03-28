INSERT INTO menus (
  parent_id, name, title, icon, path, component, type, sort, visible, status,
  api_path, api_method, created_at, updated_at
)
SELECT
  m.id,
  'alert_silences',
  '告警静默',
  'MuteNotification',
  '/monitor/silences',
  'AlertSilences',
  2,
  35,
  1,
  1,
  '/api/v1/alert-silences',
  'GET',
  NOW(),
  NOW()
FROM menus m
WHERE m.name = 'monitor_dir'
ON DUPLICATE KEY UPDATE
  title = VALUES(title),
  icon = VALUES(icon),
  path = VALUES(path),
  component = VALUES(component),
  type = VALUES(type),
  sort = VALUES(sort),
  visible = VALUES(visible),
  status = VALUES(status),
  api_path = VALUES(api_path),
  api_method = VALUES(api_method),
  updated_at = NOW();

INSERT INTO menus (
  parent_id, name, title, icon, path, component, type, sort, visible, status,
  api_path, api_method, created_at, updated_at
)
SELECT
  m.id,
  'oncall_schedules',
  'OnCall 值班',
  'Calendar',
  '/monitor/oncall',
  'OnCallSchedules',
  2,
  36,
  1,
  1,
  '/api/v1/oncall-schedules',
  'GET',
  NOW(),
  NOW()
FROM menus m
WHERE m.name = 'monitor_dir'
ON DUPLICATE KEY UPDATE
  title = VALUES(title),
  icon = VALUES(icon),
  path = VALUES(path),
  component = VALUES(component),
  type = VALUES(type),
  sort = VALUES(sort),
  visible = VALUES(visible),
  status = VALUES(status),
  api_path = VALUES(api_path),
  api_method = VALUES(api_method),
  updated_at = NOW();

INSERT IGNORE INTO role_menus (role_id, menu_id)
SELECT r.id, m.id
FROM roles r
JOIN menus m ON m.name IN ('alert_silences', 'oncall_schedules')
WHERE r.name IN ('admin', 'ops');
