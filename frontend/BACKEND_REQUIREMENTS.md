# Backend Requirements: Dashboard Refactoring

As part of the recent frontend modernization of the Dashboard (Workbench and Overview), we have introduced several new visualizations and activity streams that currently rely on mock data. 

To fully realize these features, the backend APIs need to be updated to provide the necessary data points.

## 1. Workbench API (`/api/v1/dashboard/personal`)

**Current Behavior:** 
Returns basic integer counts for personal metrics (e.g., `my_pending_tickets`, `my_assets`).

**Required Additions:**
To support the new **Sparklines** (trend visualizations) and **Pulse Status** indicators, we need historical trend data and computed status evaluations.

*   `trend_*` (Array of Integers): A list of data points representing activity over the last X periods (e.g., last 7 days). This is used to draw the Sparkline graphs.
*   `status_*` (String: `'healthy' | 'warning' | 'critical'`): A pre-computed health status for each category to drive the PulseDot indicators.

**Proposed Response Payload:**
```json
{
  "code": 0,
  "data": {
    "my_pending_tickets": 5,
    "trend_pending_tickets": [2, 3, 5, 2, 8, 4, 5],
    "status_pending_tickets": "warning",

    "my_alerts": 1,
    "trend_alerts": [0, 0, 1, 3, 0, 0, 1],
    "status_alerts": "critical",

    "my_assets": 102,
    "trend_assets": [100, 100, 101, 101, 102, 102, 102],
    "status_assets": "healthy",

    "my_task_executions": 22,
    "trend_task_executions": [10, 15, 8, 20, 25, 18, 22],
    "status_task_executions": "healthy"
  },
  "msg": "success"
}
```

## 2. Overview API (`/api/v1/dashboard/overview`)

**Current Behavior:**
Returns global system metrics (e.g., `total_users`, `total_assets`, `active_tickets`).

**Required Additions:**
To support the **System Radar** (global activity stream) and overall **System Health**, we need a feed of recent system events and a top-level health status.

*   `system_health` (String: `'healthy' | 'warning' | 'critical'`): Overall platform status.
*   `activity_stream` (Array of Objects): A list of recent notable events across the platform (tickets created, alerts triggered, deployments, etc.).

**Proposed Response Payload:**
```json
{
  "code": 0,
  "data": {
    "system_health": "healthy",
    "total_users": 150,
    "total_assets": 4500,
    "active_tickets": 23,
    
    "activity_stream": [
      {
        "id": "evt_1024",
        "type": "ticket",
        "message": "High priority ticket #1024 created by User A",
        "timestamp": "2026-03-27T10:15:00Z"
      },
      {
        "id": "evt_1025",
        "type": "alert",
        "message": "CPU usage spike on db-node-01",
        "timestamp": "2026-03-27T09:45:00Z"
      },
      {
        "id": "evt_1026",
        "type": "deploy",
        "message": "Frontend release v2.4.1 deployed successfully",
        "timestamp": "2026-03-27T08:30:00Z"
      }
    ]
  },
  "msg": "success"
}
```

## Notes
- Timeframes for trend data (`trend_*`) should be standardized (e.g., daily counts for the last 7 days).
- The `activity_stream` should be paginated or limited to the top 50 most recent events to ensure fast dashboard load times.
