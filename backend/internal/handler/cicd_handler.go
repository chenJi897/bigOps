package handler

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
	"github.com/bigops/platform/internal/service"
)

type CICDHandler struct {
	svc *service.CICDService
}

func NewCICDHandler() *CICDHandler {
	return &CICDHandler{svc: service.NewCICDService()}
}

type UpsertCICDProjectRequest struct {
	Name          string `json:"name" binding:"required"`
	Code          string `json:"code"`
	Repository    string `json:"repository"`
	RepoURL       string `json:"repo_url"`
	DefaultBranch string `json:"default_branch"`
	Description   string `json:"description"`
	OwnerID       int64  `json:"owner_id"`
	Active        *int8  `json:"active"`
	Status        *int8  `json:"status"`
}

type UpdateProjectStatusRequest struct {
	Enabled *bool `json:"enabled"`
	Active  *bool `json:"active"`
}

type UpsertCICDPipelineRequest struct {
	Name              string            `json:"name" binding:"required"`
	Code              string            `json:"code"`
	ProjectID         int64             `json:"project_id" binding:"required"`
	Description       string            `json:"description"`
	Schedule          string            `json:"schedule"`
	TriggerType       string            `json:"trigger_type"`
	TriggerRef        string            `json:"trigger_ref"`
	Branch            string            `json:"branch"`
	Environment       string            `json:"environment"`
	BuildTaskID       int64             `json:"build_task_id"`
	DeployTaskID      int64             `json:"deploy_task_id"`
	RequestTemplateID int64             `json:"request_template_id"`
	TargetHosts       []string          `json:"target_hosts"`
	BuildHosts        []string          `json:"build_hosts"`
	NotifyChannels    []string          `json:"notify_channels"`
	WebhookEnabled    *bool             `json:"webhook_enabled"`
	WebhookSecret     string            `json:"webhook_secret"`
	Variables         map[string]string `json:"variables"`
	Active            *int8             `json:"active"`
	Status            *int8             `json:"status"`
}

type TriggerPipelineRequest struct {
	Branch    string `json:"branch"`
	CommitSHA string `json:"commit_sha"`
}

type WebhookTriggerRequest struct {
	Branch        string `json:"branch"`
	CommitSHA     string `json:"commit_sha"`
	CommitMessage string `json:"commit_message"`
}

func (h *CICDHandler) ListProjects(c *gin.Context) {
	page, size := parsePageSize(c)
	var status *int8
	if raw := c.Query("active"); raw != "" {
		value, _ := strconv.ParseInt(raw, 10, 8)
		parsed := int8(value)
		status = &parsed
	}
	if raw := c.Query("status"); raw != "" {
		value, _ := strconv.ParseInt(raw, 10, 8)
		parsed := int8(value)
		status = &parsed
	}
	items, total, err := h.svc.ListProjects(repository.CICDProjectListQuery{
		Page:    page,
		Size:    size,
		Keyword: c.Query("keyword"),
		Status:  status,
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, normalizeProjects(items), total, page, size)
}

func (h *CICDHandler) CreateProject(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req UpsertCICDProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := &model.CICDProject{
		Name:          req.Name,
		Code:          req.Code,
		RepoURL:       firstNonEmpty(req.Repository, req.RepoURL),
		DefaultBranch: req.DefaultBranch,
		Description:   req.Description,
		OwnerID:       req.OwnerID,
		Status:        normalizeStatus(req.Active, req.Status),
	}
	if item.Code == "" {
		item.Code = slugifyCode(req.Name)
	}
	if err := h.svc.CreateProject(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	logger.Info("创建 CI/CD 项目", zap.String("operator", getOperator(c)), zap.String("name", item.Name))
	response.SuccessWithMessage(c, "创建成功", normalizeProject(item))
}

func (h *CICDHandler) UpdateProject(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpsertCICDProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := &model.CICDProject{
		Name:          req.Name,
		Code:          req.Code,
		RepoURL:       firstNonEmpty(req.Repository, req.RepoURL),
		DefaultBranch: req.DefaultBranch,
		Description:   req.Description,
		OwnerID:       req.OwnerID,
		Status:        normalizeStatus(req.Active, req.Status),
	}
	if item.Code == "" {
		item.Code = slugifyCode(req.Name)
	}
	if err := h.svc.UpdateProject(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

func (h *CICDHandler) DeleteProject(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeleteProject(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

func (h *CICDHandler) UpdateProjectStatus(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	project, err := h.svc.GetProject(id)
	if err != nil {
		response.Error(c, 404, "项目不存在")
		return
	}
	var req UpdateProjectStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}
	nextStatus := project.Status
	if req.Enabled != nil {
		if *req.Enabled {
			nextStatus = 1
		} else {
			nextStatus = 0
		}
	}
	if req.Active != nil {
		if *req.Active {
			nextStatus = 1
		} else {
			nextStatus = 0
		}
	}
	project.Status = nextStatus
	if err := h.svc.UpdateProject(id, project); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "状态更新成功", nil)
}

func (h *CICDHandler) ListPipelines(c *gin.Context) {
	page, size := parsePageSize(c)
	var status *int8
	if raw := c.Query("active"); raw != "" {
		value, _ := strconv.ParseInt(raw, 10, 8)
		parsed := int8(value)
		status = &parsed
	}
	if raw := c.Query("status"); raw != "" {
		value, _ := strconv.ParseInt(raw, 10, 8)
		parsed := int8(value)
		status = &parsed
	}
	projectID, _ := strconv.ParseInt(c.Query("project_id"), 10, 64)
	items, total, lastRuns, err := h.svc.ListPipelines(repository.CICDPipelineListQuery{
		Page:      page,
		Size:      size,
		ProjectID: projectID,
		Keyword:   c.Query("keyword"),
		Status:    status,
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, normalizePipelines(items, lastRuns), total, page, size)
}

func (h *CICDHandler) CreatePipeline(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	var req UpsertCICDPipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := buildPipelineModel(req)
	if err := h.svc.CreatePipeline(item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "创建成功", normalizePipeline(item, nil))
}

func (h *CICDHandler) UpdatePipeline(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req UpsertCICDPipelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}
	item := buildPipelineModel(req)
	if err := h.svc.UpdatePipeline(id, item); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "更新成功", nil)
}

func (h *CICDHandler) DeletePipeline(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := h.svc.DeletePipeline(id); err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "删除成功", nil)
}

func (h *CICDHandler) TriggerPipeline(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req TriggerPipelineRequest
	_ = c.ShouldBindJSON(&req)
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	run, err := h.svc.RunPipeline(id, operatorID, "manual", req.Branch, req.CommitSHA)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "已触发流水线", run)
}

func (h *CICDHandler) RetryRun(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	run, err := h.svc.RetryRun(id, operatorID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "已重试运行", normalizeRunSummary(run))
}

func (h *CICDHandler) RollbackRun(c *gin.Context) {
	if !requireAdmin(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID, _ := c.Get("userID")
	operatorID, _ := userID.(int64)
	run, err := h.svc.RollbackRun(id, operatorID)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "已触发回滚", normalizeRunSummary(run))
}

func (h *CICDHandler) TriggerByWebhook(c *gin.Context) {
	var req WebhookTriggerRequest
	_ = c.ShouldBindJSON(&req)
	secret := firstNonEmpty(c.GetHeader("X-BigOps-Webhook-Secret"), c.GetHeader("X-Webhook-Secret"))
	run, err := h.svc.TriggerByWebhook(c.Param("code"), secret, req.Branch, req.CommitSHA, req.CommitMessage)
	if err != nil {
		response.Error(c, 400, err.Error())
		return
	}
	response.SuccessWithMessage(c, "Webhook 已触发流水线", normalizeRunSummary(run))
}

func (h *CICDHandler) ListRuns(c *gin.Context) {
	page, size := parsePageSize(c)
	projectID, _ := strconv.ParseInt(c.Query("project_id"), 10, 64)
	pipelineID, _ := strconv.ParseInt(c.Query("pipeline_id"), 10, 64)
	items, total, err := h.svc.ListRuns(repository.CICDPipelineRunListQuery{
		Page:       page,
		Size:       size,
		ProjectID:  projectID,
		PipelineID: pipelineID,
		Status:     c.Query("status"),
	})
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	normalized := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		normalized = append(normalized, normalizeRunSummary(item))
	}
	response.Page(c, normalized, total, page, size)
}

func (h *CICDHandler) GetRunDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	run, exec, err := h.svc.GetRunDetail(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, 404, "运行记录不存在")
			return
		}
		response.InternalServerError(c, "查询失败")
		return
	}
	runPayload := normalizeRunSummary(run)
	response.Success(c, map[string]interface{}{
		"run":            runPayload,
		"task_execution": normalizeTaskExecution(exec),
	})
}

func buildPipelineModel(req UpsertCICDPipelineRequest) *model.CICDPipeline {
	targetHosts, _ := jsonMarshalStrings(req.TargetHosts)
	buildHosts, _ := jsonMarshalStrings(req.BuildHosts)
	variablesJSON, _ := jsonMarshalAny(req.Variables)
	configJSON, _ := jsonMarshalAny(map[string]interface{}{
		"webhook_enabled": boolValue(req.WebhookEnabled),
		"webhook_secret":  strings.TrimSpace(req.WebhookSecret),
		"build_hosts":     parseJSONStringSlice(buildHosts),
		"notify_channels": req.NotifyChannels,
		"variables":       req.Variables,
	})
	item := &model.CICDPipeline{
		ProjectID:         req.ProjectID,
		Name:              req.Name,
		Code:              req.Code,
		Description:       req.Description,
		TriggerType:       firstNonEmpty(req.TriggerType, req.Schedule),
		TriggerRef:        req.TriggerRef,
		Branch:            req.Branch,
		Schedule:          req.Schedule,
		Environment:       req.Environment,
		BuildTaskID:       req.BuildTaskID,
		DeployTaskID:      req.DeployTaskID,
		RequestTemplateID: req.RequestTemplateID,
		TargetHosts:       targetHosts,
		VariablesJSON:     variablesJSON,
		ConfigJSON:        configJSON,
		Status:            normalizeStatus(req.Active, req.Status),
	}
	if item.Code == "" {
		item.Code = slugifyCode(req.Name)
	}
	if item.TriggerType == "" {
		item.TriggerType = "manual"
	}
	if item.Environment == "" {
		item.Environment = "test"
	}
	if item.Branch == "" {
		item.Branch = "main"
	}
	return item
}

func normalizeProjects(items []*model.CICDProject) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, normalizeProject(item))
	}
	return result
}

func normalizeProject(item *model.CICDProject) map[string]interface{} {
	return map[string]interface{}{
		"id":             item.ID,
		"name":           item.Name,
		"code":           item.Code,
		"repository":     item.RepoURL,
		"repo_url":       item.RepoURL,
		"default_branch": item.DefaultBranch,
		"description":    item.Description,
		"owner_id":       item.OwnerID,
		"owner_name":     item.OwnerName,
		"active":         item.Status,
		"status":         item.Status,
		"created_at":     item.CreatedAt,
		"updated_at":     item.UpdatedAt,
	}
}

func normalizePipelines(items []*model.CICDPipeline, lastRuns map[int64]*model.CICDPipelineRun) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, normalizePipeline(item, lastRuns[item.ID]))
	}
	return result
}

func normalizePipeline(item *model.CICDPipeline, lastRun *model.CICDPipelineRun) map[string]interface{} {
	config := parseJSONStringToMap(item.ConfigJSON)
	variables := parseJSONStringToMap(item.VariablesJSON)
	buildHosts := parseInterfaceSliceToStrings(config["build_hosts"])
	if len(buildHosts) == 0 {
		buildHosts = parseJSONStringSlice(item.TargetHosts)
	}
	result := map[string]interface{}{
		"id":                    item.ID,
		"name":                  item.Name,
		"code":                  item.Code,
		"project_id":            item.ProjectID,
		"project_name":          item.ProjectName,
		"description":           item.Description,
		"schedule":              item.Schedule,
		"trigger_type":          item.TriggerType,
		"branch":                item.Branch,
		"environment":           item.Environment,
		"build_task_id":         item.BuildTaskID,
		"build_task_name":       item.BuildTaskName,
		"deploy_task_id":        item.DeployTaskID,
		"deploy_task_name":      item.DeployTaskName,
		"request_template_id":   item.RequestTemplateID,
		"request_template_name": item.RequestTemplateName,
		"target_hosts":          item.TargetHosts,
		"target_hosts_list":     parseJSONStringSlice(item.TargetHosts),
		"variables_json":        item.VariablesJSON,
		"variables":             variables,
		"config_json":           item.ConfigJSON,
		"webhook_enabled":       parseBoolValue(config["webhook_enabled"]),
		"webhook_secret":        parseStringValue(config["webhook_secret"]),
		"notify_channels":       parseInterfaceSliceToStrings(config["notify_channels"]),
		"webhook_code":          item.Code,
		"webhook_path":          "/api/v1/cicd/webhook/" + item.Code,
		"build_hosts_list":      buildHosts,
		"active":                item.Status,
		"status":                item.Status,
		"created_at":            item.CreatedAt,
		"updated_at":            item.UpdatedAt,
	}
	latestRun := normalizeRunSummary(lastRun)
	result["latest_run"] = latestRun
	if lastRun != nil {
		result["latest_run_status"] = lastRun.Status
		result["latest_run_number"] = lastRun.RunNumber
		result["latest_run_result"] = lastRun.Result
	} else {
		result["latest_run_status"] = nil
		result["latest_run_number"] = nil
		result["latest_run_result"] = nil
	}
	return result
}

func normalizeRunSummary(run *model.CICDPipelineRun) map[string]interface{} {
	if run == nil {
		return nil
	}
	artifact := parseJSONStringToMap(run.ArtifactSummary)
	buildMap := parseInterfaceToMap(artifact["build"])
	approvalMap := parseInterfaceToMap(artifact["approval"])
	deployMap := parseInterfaceToMap(artifact["deploy"])
	return map[string]interface{}{
		"id":                       run.ID,
		"run_number":               run.RunNumber,
		"status":                   run.Status,
		"result":                   run.Result,
		"summary":                  run.Summary,
		"error_message":            run.ErrorMessage,
		"trigger_type":             run.TriggerType,
		"trigger_ref":              run.TriggerRef,
		"branch":                   run.Branch,
		"project_id":               run.ProjectID,
		"project_name":             run.ProjectName,
		"pipeline_id":              run.PipelineID,
		"pipeline_name":            run.PipelineName,
		"started_at":               run.StartedAt,
		"finished_at":              run.FinishedAt,
		"duration_seconds":         run.DurationSeconds,
		"queued_seconds":           run.QueuedSeconds,
		"triggered_by":             run.TriggeredBy,
		"triggered_by_name":        run.TriggeredByName,
		"task_execution_id":        run.TaskExecutionID,
		"approval_ticket_id":       run.ApprovalTicketID,
		"artifact_summary":         run.ArtifactSummary,
		"target_hosts":             run.TargetHosts,
		"commit_id":                run.CommitID,
		"commit_message":           run.CommitMessage,
		"variables_json":           run.VariablesJSON,
		"metadata_json":            run.MetadataJSON,
		"log_snippet":              run.LogSnippet,
		"target_hosts_list":        parseJSONStringSlice(run.TargetHosts),
		"variables":                parseJSONStringToMap(run.VariablesJSON),
		"metadata":                 parseJSONStringToMap(run.MetadataJSON),
		"artifact_summary_map":     artifact,
		"current_stage":            artifact["current_stage"],
		"build_stage_status":       buildMap["status"],
		"build_status":             buildMap["status"],
		"build_summary":            buildMap["summary"],
		"build_error":              buildMap["error"],
		"approval_stage_status":    approvalMap["status"],
		"approval_status":          approvalMap["status"],
		"approval_summary":         approvalMap["summary"],
		"approval_error":           approvalMap["error"],
		"deploy_stage_status":      deployMap["status"],
		"deploy_status":            deployMap["status"],
		"deploy_summary":           deployMap["summary"],
		"deploy_error":             deployMap["error"],
		"build_execution_id":       buildMap["execution_id"],
		"deploy_execution_id":      deployMap["execution_id"],
		"approval_ticket_id_stage": approvalMap["ticket_id"],
		"webhook_enabled":          artifact["webhook_enabled"],
		"build_hosts_list":         artifact["build_hosts_list"],
		"pipeline_variables":       artifact["variables"],
	}
}

func normalizeTaskExecution(exec *model.TaskExecution) map[string]interface{} {
	if exec == nil {
		return nil
	}
	return map[string]interface{}{
		"id":                exec.ID,
		"task_id":           exec.TaskID,
		"task_name":         exec.TaskName,
		"status":            exec.Status,
		"target_hosts":      exec.TargetHosts,
		"target_hosts_list": parseJSONStringSlice(exec.TargetHosts),
		"total_count":       exec.TotalCount,
		"success_count":     exec.SuccessCount,
		"fail_count":        exec.FailCount,
		"operator_id":       exec.OperatorID,
		"operator_name":     exec.OperatorName,
		"started_at":        exec.StartedAt,
		"finished_at":       exec.FinishedAt,
		"created_at":        exec.CreatedAt,
		"updated_at":        exec.UpdatedAt,
		"host_results":      exec.HostResults,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeStatus(active *int8, status *int8) int8 {
	if active != nil {
		return *active
	}
	if status != nil {
		return *status
	}
	return 1
}

func slugifyCode(name string) string {
	result := make([]rune, 0, len(name))
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			result = append(result, r)
			lastDash = false
		case !lastDash && len(result) > 0:
			result = append(result, '-')
			lastDash = true
		}
	}
	code := strings.Trim(string(result), "-")
	if code == "" {
		return "cicd"
	}
	return code
}

func jsonMarshalStrings(items []string) (string, error) {
	if len(items) == 0 {
		return "[]", nil
	}
	data, err := json.Marshal(items)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func jsonMarshalAny(value interface{}) (string, error) {
	if value == nil {
		return "{}", nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func boolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func parseJSONStringSlice(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func parseJSONStringToMap(raw string) map[string]interface{} {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil
	}
	return data
}

func parseInterfaceToMap(value interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	result, _ := value.(map[string]interface{})
	return result
}

func parseInterfaceSliceToStrings(value interface{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		result = append(result, parseStringValue(item))
	}
	return result
}

func parseStringValue(value interface{}) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}

func parseBoolValue(value interface{}) bool {
	if value == nil {
		return false
	}
	if v, ok := value.(bool); ok {
		return v
	}
	return false
}
