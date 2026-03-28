package service

import (
	"testing"

	"github.com/bigops/platform/internal/model"
)

func TestValidateAlertRuleActionCreateTicket(t *testing.T) {
	rule := &model.AlertRule{
		Name:         "cpu high",
		MetricType:   "cpu_usage",
		Operator:     "gt",
		Threshold:    90,
		Severity:     "critical",
		Action:       model.AlertRuleActionCreateTicket,
		TicketTypeID: 1,
	}
	if err := validateAlertRule(rule); err != nil {
		t.Fatalf("expected create_ticket to pass, got %v", err)
	}
}

func TestValidateAlertRuleActionCreateTicketWithoutTypeID(t *testing.T) {
	rule := &model.AlertRule{
		Name:       "cpu high no type",
		MetricType: "cpu_usage",
		Operator:   "gt",
		Threshold:  90,
		Severity:   "warning",
		Action:     model.AlertRuleActionCreateTicket,
	}
	if err := validateAlertRule(rule); err != nil {
		t.Fatalf("expected create_ticket without ticket type to pass, got %v", err)
	}
}

func TestValidateAlertRuleActionExecuteTaskMissingTask(t *testing.T) {
	rule := &model.AlertRule{
		Name:       "cpu high",
		MetricType: "cpu_usage",
		Operator:   "gt",
		Threshold:  90,
		Severity:   "critical",
		Action:     model.AlertRuleActionExecuteTask,
	}
	if err := validateAlertRule(rule); err == nil {
		t.Fatalf("expected execute_task without repair task to fail")
	}
}
