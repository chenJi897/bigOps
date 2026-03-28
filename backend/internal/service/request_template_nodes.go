package service

import (
	"encoding/json"

	"github.com/bigops/platform/internal/model"
)

type requestTemplateNode struct {
	NodeID             string  `json:"node_id"`
	Name               string  `json:"name"`
	ApproveMode        string  `json:"approve_mode"`
	HandlerIDs         []int64 `json:"handler_ids"`
	OptionalHandlerIDs []int64 `json:"optional_handler_ids"`
	NotifyUserIDs      []int64 `json:"notify_user_ids"`
	NodeFormSchema     string  `json:"node_form_schema"`
	CallbackConfig     string  `json:"callback_config"`
	Sort               int     `json:"sort"`
}

func parseRequestTemplateNodes(raw string) []requestTemplateNode {
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []requestTemplateNode
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func buildApprovalStagesFromTemplate(template *model.RequestTemplate) []model.ApprovalPolicyStage {
	nodes := parseRequestTemplateNodes(template.NodesJSON)
	stages := make([]model.ApprovalPolicyStage, 0, len(nodes))
	stageNo := 1
	for _, node := range nodes {
		if node.ApproveMode == "none" {
			continue
		}
		userIDs := node.HandlerIDs
		if len(userIDs) == 0 {
			userIDs = node.OptionalHandlerIDs
		}
		if len(userIDs) == 0 {
			continue
		}
		passRule := "any"
		if node.ApproveMode == "and" {
			passRule = "all"
		}
		config, _ := json.Marshal(map[string]interface{}{
			"user_ids": userIDs,
		})
		stageName := node.Name
		if stageName == "" {
			stageName = "审批"
		}
		stages = append(stages, model.ApprovalPolicyStage{
			StageNo:        stageNo,
			Name:           stageName,
			StageType:      "serial",
			ApproverType:   "fixed_user",
			ApproverConfig: string(config),
			PassRule:       passRule,
			TimeoutHours:   24,
			Required:       1,
			Sort:           node.Sort,
		})
		stageNo++
	}
	return stages
}
