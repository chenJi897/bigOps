package service

import (
	"testing"

	"github.com/bigops/platform/internal/model"
)

func TestParsePipelineRuntimeConfig_BuildHostsFallbackAndWebhook(t *testing.T) {
	targetHosts := []string{"10.0.0.11", "10.0.0.12"}
	cfg := parsePipelineRuntimeConfig(
		`{"webhook_enabled":true,"webhook_secret":"abc","variables":{"A":"1"}}`,
		`{"B":"2"}`,
		targetHosts,
	)

	if !cfg.WebhookEnabled {
		t.Fatalf("expected webhook enabled")
	}
	if cfg.WebhookSecret != "abc" {
		t.Fatalf("expected webhook secret abc, got %q", cfg.WebhookSecret)
	}
	if len(cfg.BuildHosts) != 2 || cfg.BuildHosts[0] != "10.0.0.11" {
		t.Fatalf("expected build hosts fallback to target hosts, got %#v", cfg.BuildHosts)
	}
	if cfg.Variables["A"] != "1" || cfg.Variables["B"] != "2" {
		t.Fatalf("expected merged variables, got %#v", cfg.Variables)
	}
}

func TestParsePipelineRuntimeConfig_PreferConfigBuildHosts(t *testing.T) {
	cfg := parsePipelineRuntimeConfig(
		`{"build_hosts":["192.168.1.10","192.168.1.11"]}`,
		`{}`,
		[]string{"10.0.0.11"},
	)
	if len(cfg.BuildHosts) != 2 || cfg.BuildHosts[0] != "192.168.1.10" {
		t.Fatalf("expected config build hosts, got %#v", cfg.BuildHosts)
	}
}

func TestBuildTaskEnv_MergesStageAndSourceRunID(t *testing.T) {
	svc := &CICDService{}
	project := mockProject
	pipeline := mockPipeline
	run := mockRun
	env := svc.buildTaskEnv(project, pipeline, run, "deploy", map[string]string{"X": "y"})

	if env["CICD_STAGE"] != "deploy" {
		t.Fatalf("expected stage deploy, got %q", env["CICD_STAGE"])
	}
	if env["CICD_SOURCE_RUN_ID"] != "88" {
		t.Fatalf("expected source run id 88, got %q", env["CICD_SOURCE_RUN_ID"])
	}
	if env["GLOBAL"] != "g" || env["X"] != "y" {
		t.Fatalf("expected merged env vars, got %#v", env)
	}
}

func TestShouldNotifyPipelineStatusTransition(t *testing.T) {
	tests := []struct {
		name       string
		before     string
		after      string
		wantNotify bool
	}{
		{name: "pending to success notifies", before: "pending", after: "success", wantNotify: true},
		{name: "running to failed notifies", before: "running", after: "failed", wantNotify: true},
		{name: "running to canceled notifies", before: "running", after: "canceled", wantNotify: true},
		{name: "success to success no duplicate", before: "success", after: "success", wantNotify: false},
		{name: "failed to failed no duplicate", before: "failed", after: "failed", wantNotify: false},
		{name: "canceled to canceled no duplicate", before: "canceled", after: "canceled", wantNotify: false},
		{name: "running to running no notify", before: "running", after: "running", wantNotify: false},
		{name: "pending to running no notify", before: "pending", after: "running", wantNotify: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldNotifyPipelineStatusTransition(tt.before, tt.after)
			if got != tt.wantNotify {
				t.Fatalf("expected %v, got %v", tt.wantNotify, got)
			}
		})
	}
}

var mockProject = modelCICDProject()
var mockPipeline = modelCICDPipeline()
var mockRun = modelCICDPipelineRun()

func modelCICDProject() *model.CICDProject {
	return &model.CICDProject{
		ID:   9,
		Name: "proj-a",
	}
}

func modelCICDPipeline() *model.CICDPipeline {
	return &model.CICDPipeline{
		ID:            5,
		Name:          "pipe-a",
		Environment:   "prod",
		VariablesJSON: `{"GLOBAL":"g"}`,
	}
}

func modelCICDPipelineRun() *model.CICDPipelineRun {
	return &model.CICDPipelineRun{
		ID:           101,
		RunNumber:    7,
		TriggerType:  "manual",
		Branch:       "main",
		CommitID:     "sha1",
		MetadataJSON: `{"source_run_id":88}`,
	}
}
