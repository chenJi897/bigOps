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
