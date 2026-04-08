package middleware

import "testing"

func TestShouldBypassCasbin(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		method string
		want   bool
	}{
		{name: "auth logout post allowed", path: "/api/v1/auth/logout", method: "POST", want: true},
		{name: "departments all get allowed", path: "/api/v1/departments/all", method: "GET", want: true},
		{name: "notifications preferences post blocked", path: "/api/v1/notifications/preferences", method: "POST", want: false},
		{name: "monitor summary should not bypass", path: "/api/v1/monitor/summary", method: "GET", want: false},
		{name: "tasks list should not bypass", path: "/api/v1/tasks", method: "GET", want: false},
		{name: "cicd runs should not bypass", path: "/api/v1/cicd/runs", method: "GET", want: false},
		{name: "websocket prefix allowed", path: "/api/v1/ws/task-executions/1/logs", method: "GET", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldBypassCasbin(tt.path, tt.method)
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
