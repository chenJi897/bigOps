package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildInspectionReportCSV(t *testing.T) {
	report := map[string]interface{}{
		"record_id": 101,
		"status":    "success",
		"detail": map[string]interface{}{
			"template_name": "nightly-check",
			"target_hosts":  []string{"10.0.0.1"},
		},
	}
	payload, err := buildInspectionReportCSV(report)
	if err != nil {
		t.Fatalf("buildInspectionReportCSV returned error: %v", err)
	}
	content := string(payload)
	if !strings.Contains(content, "field,value") {
		t.Fatalf("csv header missing, got: %s", content)
	}
	if !strings.Contains(content, "record_id,101") {
		t.Fatalf("record_id row missing, got: %s", content)
	}
}

func TestBuildInspectionReportCSVJSONValue(t *testing.T) {
	report := map[string]interface{}{
		"detail": map[string]interface{}{"k": "v"},
	}
	payload, err := buildInspectionReportCSV(report)
	if err != nil {
		t.Fatalf("buildInspectionReportCSV returned error: %v", err)
	}
	if !strings.Contains(string(payload), `"{""k"":""v""}"`) {
		t.Fatalf("expected json encoded map value in csv, got: %s", string(payload))
	}

	var m map[string]interface{}
	if err := json.Unmarshal([]byte(`{"k":"v"}`), &m); err != nil {
		t.Fatalf("json unmarshal sanity check failed: %v", err)
	}
}
