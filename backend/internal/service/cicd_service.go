package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type CICDService struct {
	projectRepo   *repository.CICDProjectRepository
	pipelineRepo  *repository.CICDPipelineRepository
	runRepo       *repository.CICDPipelineRunRepository
	userRepo      *repository.UserRepository
	taskRepo      *repository.TaskRepository
	taskExecRepo  *repository.TaskExecutionRepository
	requestRepo   *repository.RequestTemplateRepository
	ticketService *TicketService
	taskService   *TaskService
	notifySvc     *NotificationService
}

type CICDTriggerInput struct {
	TriggerType string
	Branch      string
	CommitSHA   string
	CommitMsg   string
	TriggerRef  string
	SourceRunID int64
}

type pipelineRuntimeConfig struct {
	WebhookEnabled bool
	WebhookSecret  string
	BuildHosts     []string
	Variables      map[string]string
	NotifyChannels []string
}

type cicdStageState struct {
	TaskID      int64    `json:"task_id,omitempty"`
	ExecutionID int64    `json:"execution_id,omitempty"`
	Status      string   `json:"status,omitempty"`
	Hosts       []string `json:"hosts,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Error       string   `json:"error,omitempty"`
}

type cicdApprovalStageState struct {
	Required bool   `json:"required"`
	TicketID int64  `json:"ticket_id,omitempty"`
	Status   string `json:"status,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Error    string `json:"error,omitempty"`
}

type cicdRunStageSummary struct {
	CurrentStage   string                 `json:"current_stage,omitempty"`
	WebhookEnabled bool                   `json:"webhook_enabled"`
	BuildHostsList []string               `json:"build_hosts_list,omitempty"`
	Variables      map[string]string      `json:"variables,omitempty"`
	Build          cicdStageState         `json:"build"`
	Approval       cicdApprovalStageState `json:"approval"`
	Deploy         cicdStageState         `json:"deploy"`
}

func NewCICDService() *CICDService {
	return &CICDService{
		projectRepo:   repository.NewCICDProjectRepository(),
		pipelineRepo:  repository.NewCICDPipelineRepository(),
		runRepo:       repository.NewCICDPipelineRunRepository(),
		userRepo:      repository.NewUserRepository(),
		taskRepo:      repository.NewTaskRepository(),
		taskExecRepo:  repository.NewTaskExecutionRepository(),
		requestRepo:   repository.NewRequestTemplateRepository(),
		ticketService: NewTicketService(),
		taskService:   NewTaskService(),
		notifySvc:     NewNotificationService(),
	}
}

func (s *CICDService) ListProjects(q repository.CICDProjectListQuery) ([]*model.CICDProject, int64, error) {
	items, total, err := s.projectRepo.List(q)
	if err != nil {
		return nil, 0, err
	}
	s.fillProjectOwners(items)
	return items, total, nil
}

func (s *CICDService) CreateProject(item *model.CICDProject) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("项目名称不能为空")
	}
	if strings.TrimSpace(item.Code) == "" {
		return errors.New("项目编码不能为空")
	}
	if strings.TrimSpace(item.RepoURL) == "" {
		return errors.New("仓库地址不能为空")
	}
	if _, err := s.projectRepo.GetByName(item.Name); err == nil {
		return errors.New("项目名称已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if _, err := s.projectRepo.GetByCode(item.Code); err == nil {
		return errors.New("项目编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if item.DefaultBranch == "" {
		item.DefaultBranch = "main"
	}
	return s.projectRepo.Create(item)
}

func (s *CICDService) UpdateProject(id int64, item *model.CICDProject) error {
	existing, err := s.projectRepo.GetByID(id)
	if err != nil {
		return errors.New("项目不存在")
	}
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("项目名称不能为空")
	}
	if strings.TrimSpace(item.Code) == "" {
		return errors.New("项目编码不能为空")
	}
	if strings.TrimSpace(item.RepoURL) == "" {
		return errors.New("仓库地址不能为空")
	}
	if item.Name != existing.Name {
		if dup, err := s.projectRepo.GetByName(item.Name); err == nil && dup.ID != id {
			return errors.New("项目名称已存在")
		}
	}
	if item.Code != existing.Code {
		if dup, err := s.projectRepo.GetByCode(item.Code); err == nil && dup.ID != id {
			return errors.New("项目编码已存在")
		}
	}
	existing.Name = item.Name
	existing.Code = item.Code
	existing.RepoProvider = item.RepoProvider
	existing.RepoURL = item.RepoURL
	existing.DefaultBranch = item.DefaultBranch
	existing.Description = item.Description
	existing.OwnerID = item.OwnerID
	existing.Status = item.Status
	return s.projectRepo.Update(existing)
}

func (s *CICDService) DeleteProject(id int64) error {
	if _, err := s.projectRepo.GetByID(id); err != nil {
		return errors.New("项目不存在")
	}
	return s.projectRepo.Delete(id)
}

func (s *CICDService) GetProject(id int64) (*model.CICDProject, error) {
	item, err := s.projectRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillProjectOwners([]*model.CICDProject{item})
	return item, nil
}

func (s *CICDService) ListPipelines(q repository.CICDPipelineListQuery) ([]*model.CICDPipeline, int64, map[int64]*model.CICDPipelineRun, error) {
	items, total, err := s.pipelineRepo.List(q)
	if err != nil {
		return nil, 0, nil, err
	}
	s.fillPipelines(items)
	lastRuns := make(map[int64]*model.CICDPipelineRun)
	runs := make([]*model.CICDPipelineRun, 0, len(items))
	for _, item := range items {
		run, err := s.runRepo.GetLatestByPipeline(item.ID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, 0, nil, err
		}
		_ = s.syncRunExecutionStatus(run)
		lastRuns[item.ID] = run
		runs = append(runs, run)
	}
	if len(runs) > 0 {
		s.fillRuns(runs)
	}
	return items, total, lastRuns, nil
}

func (s *CICDService) CreatePipeline(item *model.CICDPipeline) error {
	if item.ProjectID == 0 {
		return errors.New("请选择所属项目")
	}
	if _, err := s.projectRepo.GetByID(item.ProjectID); err != nil {
		return errors.New("所属项目不存在")
	}
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("流水线名称不能为空")
	}
	if strings.TrimSpace(item.Code) == "" {
		return errors.New("流水线编码不能为空")
	}
	if _, err := s.pipelineRepo.GetByName(item.Name); err == nil {
		return errors.New("流水线名称已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if _, err := s.pipelineRepo.GetByCode(item.Code); err == nil {
		return errors.New("流水线编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := s.validatePipelineRefs(item); err != nil {
		return err
	}
	if item.Environment == "" {
		item.Environment = "test"
	}
	if item.Branch == "" {
		item.Branch = "main"
	}
	item.Status = defaultPipelineStatus(item.Status)
	return s.pipelineRepo.Create(item)
}

func (s *CICDService) UpdatePipeline(id int64, item *model.CICDPipeline) error {
	existing, err := s.pipelineRepo.GetByID(id)
	if err != nil {
		return errors.New("流水线不存在")
	}
	if item.ProjectID == 0 {
		return errors.New("请选择所属项目")
	}
	if _, err := s.projectRepo.GetByID(item.ProjectID); err != nil {
		return errors.New("所属项目不存在")
	}
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("流水线名称不能为空")
	}
	if strings.TrimSpace(item.Code) == "" {
		return errors.New("流水线编码不能为空")
	}
	if item.Name != existing.Name {
		if dup, err := s.pipelineRepo.GetByName(item.Name); err == nil && dup.ID != id {
			return errors.New("流水线名称已存在")
		}
	}
	if item.Code != existing.Code {
		if dup, err := s.pipelineRepo.GetByCode(item.Code); err == nil && dup.ID != id {
			return errors.New("流水线编码已存在")
		}
	}
	if err := s.validatePipelineRefs(item); err != nil {
		return err
	}
	existing.ProjectID = item.ProjectID
	existing.Name = item.Name
	existing.Code = item.Code
	existing.TriggerType = item.TriggerType
	existing.TriggerRef = item.TriggerRef
	existing.Schedule = item.Schedule
	existing.Branch = item.Branch
	existing.Environment = item.Environment
	existing.BuildTaskID = item.BuildTaskID
	existing.DeployTaskID = item.DeployTaskID
	existing.RequestTemplateID = item.RequestTemplateID
	existing.TargetHosts = item.TargetHosts
	existing.VariablesJSON = item.VariablesJSON
	existing.ConfigJSON = item.ConfigJSON
	existing.Description = item.Description
	existing.Status = item.Status
	return s.pipelineRepo.Update(existing)
}

func defaultPipelineStatus(status int8) int8 {
	if status == 0 {
		return 1
	}
	return status
}

func (s *CICDService) DeletePipeline(id int64) error {
	if _, err := s.pipelineRepo.GetByID(id); err != nil {
		return errors.New("流水线不存在")
	}
	return s.pipelineRepo.Delete(id)
}

func (s *CICDService) ListRuns(q repository.CICDPipelineRunListQuery) ([]*model.CICDPipelineRun, int64, error) {
	items, total, err := s.runRepo.List(q)
	if err != nil {
		return nil, 0, err
	}
	for _, item := range items {
		_ = s.syncRunExecutionStatus(item)
	}
	s.fillRuns(items)
	return items, total, nil
}

func (s *CICDService) GetRunDetail(id int64) (*model.CICDPipelineRun, *model.TaskExecution, error) {
	run, err := s.runRepo.GetByID(id)
	if err != nil {
		return nil, nil, err
	}
	if err := s.syncRunExecutionStatus(run); err == nil {
		if refreshed, refreshErr := s.runRepo.GetByID(id); refreshErr == nil {
			run = refreshed
		}
	}
	var exec *model.TaskExecution
	if run.TaskExecutionID > 0 {
		exec, err = s.taskExecRepo.GetByID(run.TaskExecutionID)
		if err != nil {
			return run, nil, err
		}
		exec.OperatorName = s.getUserName(exec.OperatorID)
		if hostResults, err := s.taskExecRepo.GetHostResultsByExecutionID(exec.ID); err == nil {
			exec.HostResults = make([]model.TaskHostResult, 0, len(hostResults))
			for _, item := range hostResults {
				if item == nil {
					continue
				}
				exec.HostResults = append(exec.HostResults, *item)
			}
		}
	}
	s.fillRuns([]*model.CICDPipelineRun{run})
	return run, exec, nil
}

func (s *CICDService) RunPipeline(pipelineID, operatorID int64, triggerType, branch, commitSHA string) (*model.CICDPipelineRun, error) {
	return s.runPipelineInternal(pipelineID, operatorID, CICDTriggerInput{
		TriggerType: triggerType,
		Branch:      branch,
		CommitSHA:   commitSHA,
		TriggerRef:  branch,
	})
}

func (s *CICDService) RetryRun(runID, operatorID int64) (*model.CICDPipelineRun, error) {
	existing, err := s.runRepo.GetByID(runID)
	if err != nil {
		return nil, errors.New("运行记录不存在")
	}
	return s.runPipelineInternal(existing.PipelineID, operatorID, CICDTriggerInput{
		TriggerType: "retry",
		Branch:      existing.Branch,
		CommitSHA:   existing.CommitID,
		CommitMsg:   existing.CommitMessage,
		TriggerRef:  existing.TriggerRef,
		SourceRunID: existing.ID,
	})
}

func (s *CICDService) RollbackRun(runID, operatorID int64) (*model.CICDPipelineRun, error) {
	existing, err := s.runRepo.GetByID(runID)
	if err != nil {
		return nil, errors.New("运行记录不存在")
	}
	return s.runPipelineInternal(existing.PipelineID, operatorID, CICDTriggerInput{
		TriggerType: "rollback",
		Branch:      existing.Branch,
		CommitSHA:   existing.CommitID,
		CommitMsg:   existing.CommitMessage,
		TriggerRef:  fmt.Sprintf("rollback:%d", existing.ID),
		SourceRunID: existing.ID,
	})
}

func (s *CICDService) TriggerByWebhook(pipelineCode, providedSecret, branch, commitSHA, commitMessage string) (*model.CICDPipelineRun, error) {
	pipeline, err := s.pipelineRepo.GetByCode(pipelineCode)
	if err != nil {
		return nil, errors.New("流水线不存在")
	}
	runtimeCfg := parsePipelineRuntimeConfig(pipeline.ConfigJSON, pipeline.VariablesJSON, parseTargetHosts(pipeline.TargetHosts))
	if !runtimeCfg.WebhookEnabled {
		return nil, errors.New("该流水线未启用 webhook 触发")
	}
	if strings.TrimSpace(runtimeCfg.WebhookSecret) != "" && strings.TrimSpace(providedSecret) != strings.TrimSpace(runtimeCfg.WebhookSecret) {
		return nil, errors.New("webhook secret 校验失败")
	}
	return s.runPipelineInternal(pipeline.ID, 0, CICDTriggerInput{
		TriggerType: "webhook",
		Branch:      branch,
		CommitSHA:   commitSHA,
		CommitMsg:   commitMessage,
		TriggerRef:  branch,
	})
}

func (s *CICDService) runPipelineInternal(pipelineID, operatorID int64, input CICDTriggerInput) (*model.CICDPipelineRun, error) {
	pipeline, err := s.pipelineRepo.GetByID(pipelineID)
	if err != nil {
		return nil, errors.New("流水线不存在")
	}
	if pipeline.Status != 1 {
		return nil, errors.New("流水线已禁用")
	}
	project, err := s.projectRepo.GetByID(pipeline.ProjectID)
	if err != nil {
		return nil, errors.New("所属项目不存在")
	}
	if input.TriggerType == "" {
		input.TriggerType = "manual"
	}
	if input.Branch == "" {
		input.Branch = firstNonEmpty(pipeline.Branch, pipeline.TriggerRef, project.DefaultBranch)
	}
	targetHosts := parseTargetHosts(pipeline.TargetHosts)
	runtimeCfg := parsePipelineRuntimeConfig(pipeline.ConfigJSON, pipeline.VariablesJSON, targetHosts)
	buildHosts := runtimeCfg.BuildHosts
	buildRequired := pipeline.BuildTaskID > 0 && len(buildHosts) > 0
	deployRequired := pipeline.DeployTaskID > 0 && len(targetHosts) > 0
	approvalRequired := pipeline.RequestTemplateID > 0
	runNumber := int64(1)
	if latest, err := s.runRepo.GetLatestByPipeline(pipeline.ID); err == nil {
		runNumber = latest.RunNumber + 1
	}
	stages := cicdRunStageSummary{
		WebhookEnabled: runtimeCfg.WebhookEnabled,
		BuildHostsList: buildHosts,
		Variables:      runtimeCfg.Variables,
		Build: cicdStageState{
			TaskID: pipeline.BuildTaskID,
			Hosts:  buildHosts,
		},
		Approval: cicdApprovalStageState{
			Required: approvalRequired,
		},
		Deploy: cicdStageState{
			TaskID: pipeline.DeployTaskID,
			Hosts:  targetHosts,
		},
	}
	switch {
	case buildRequired:
		stages.CurrentStage = "build"
		stages.Build.Status = "pending"
	case approvalRequired:
		stages.CurrentStage = "approval"
		stages.Build.Status = "skipped"
		stages.Approval.Status = "pending"
	case deployRequired:
		stages.CurrentStage = "deploy"
		stages.Build.Status = "skipped"
		stages.Approval.Status = "skipped"
		stages.Deploy.Status = "pending"
	default:
		stages.CurrentStage = "done"
		stages.Build.Status = "skipped"
		stages.Approval.Status = "skipped"
		stages.Deploy.Status = "skipped"
	}

	run := &model.CICDPipelineRun{
		ProjectID:       pipeline.ProjectID,
		PipelineID:      pipeline.ID,
		RunNumber:       runNumber,
		TriggerType:     input.TriggerType,
		TriggerRef:      firstNonEmpty(input.TriggerRef, input.Branch, pipeline.TriggerRef),
		Branch:          input.Branch,
		CommitID:        input.CommitSHA,
		CommitMessage:   input.CommitMsg,
		Status:          "pending",
		TriggeredBy:     operatorID,
		TargetHosts:     pipeline.TargetHosts,
		VariablesJSON:   mustMarshalMapString(runtimeCfg.Variables),
		ArtifactSummary: mustMarshalRunStages(stages),
		Summary:         fmt.Sprintf("触发流水线：%s", pipeline.Name),
	}
	if input.SourceRunID > 0 {
		run.MetadataJSON = fmt.Sprintf(`{"source_run_id":%d}`, input.SourceRunID)
	}
	now := model.LocalTime(time.Now())
	run.StartedAt = now
	run.FinishedAt = now

	if err := s.runRepo.Create(run); err != nil {
		return nil, err
	}

	if buildRequired {
		exec, err := s.taskService.ExecuteTaskWithEnv(
			pipeline.BuildTaskID,
			buildHosts,
			operatorID,
			s.buildTaskEnv(project, pipeline, run, "build", nil),
		)
		if err != nil {
			stages = parseRunStages(run.ArtifactSummary)
			stages.Build.Status = "failed"
			stages.Build.Error = err.Error()
			stages.Build.Summary = "构建触发失败"
			stages.CurrentStage = "done"
			run.ArtifactSummary = mustMarshalRunStages(stages)
			run.Status = "failed"
			run.Result = "build_failed"
			run.ErrorMessage = err.Error()
			run.Summary = "构建触发失败：" + err.Error()
		} else {
			stages = parseRunStages(run.ArtifactSummary)
			stages.Build.ExecutionID = exec.ID
			stages.Build.Status = "running"
			stages.Build.Summary = fmt.Sprintf("已触发构建任务 #%d", exec.ID)
			stages.CurrentStage = "build"
			run.ArtifactSummary = mustMarshalRunStages(stages)
			run.TaskExecutionID = exec.ID
			run.Status = "running"
			run.Summary = fmt.Sprintf("已触发构建任务 #%d", exec.ID)
		}
	} else {
		if err := s.advanceAfterBuildSuccess(run, pipeline, project, targetHosts); err != nil {
			return nil, err
		}
	}

	if run.Status == "pending" {
		run.Status = "created"
	}
	if err := s.saveRun(run); err != nil {
		return nil, err
	}
	s.fillRuns([]*model.CICDPipelineRun{run})
	return run, nil
}

func (s *CICDService) advanceAfterBuildSuccess(run *model.CICDPipelineRun, pipeline *model.CICDPipeline, project *model.CICDProject, deployHosts []string) error {
	if run == nil || pipeline == nil {
		return nil
	}
	stages := parseRunStages(run.ArtifactSummary)
	if stages.Build.TaskID > 0 {
		stages.Build.Status = "success"
		stages.Build.Summary = "构建阶段完成"
		stages.Build.Error = ""
	}
	if stages.Approval.Required {
		if run.ApprovalTicketID > 0 || stages.Approval.TicketID > 0 {
			stages.Approval.Status = "waiting"
			stages.CurrentStage = "approval"
			stages.Approval.Summary = fmt.Sprintf("等待审批工单 #%d", firstNonZero(run.ApprovalTicketID, stages.Approval.TicketID))
			run.ArtifactSummary = mustMarshalRunStages(stages)
			run.Status = "waiting_approval"
			run.TaskExecutionID = 0
			if run.Summary == "" {
				run.Summary = stages.Approval.Summary
			}
			return nil
		}
		ticket := &model.Ticket{
			Title:             fmt.Sprintf("发布审批：%s", pipeline.Name),
			Description:       fmt.Sprintf("项目 %s / 环境 %s / 分支 %s", project.Name, pipeline.Environment, run.Branch),
			RequestTemplateID: pipeline.RequestTemplateID,
			TicketKind:        "change",
		}
		if err := s.ticketService.Create(ticket, run.TriggeredBy, s.getUserName(run.TriggeredBy)); err != nil {
			return err
		}
		run.ApprovalTicketID = ticket.ID
		stages.Approval.TicketID = ticket.ID
		stages.Approval.Status = "waiting"
		stages.Approval.Summary = fmt.Sprintf("等待审批工单 #%d", ticket.ID)
		stages.Approval.Error = ""
		stages.CurrentStage = "approval"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.Status = "waiting_approval"
		run.TaskExecutionID = 0
		run.Summary = fmt.Sprintf("构建完成，等待审批工单 #%d", ticket.ID)
		return nil
	}
	stages.Approval.Status = "skipped"
	stages.Approval.Summary = "无需审批"
	if stages.Deploy.TaskID > 0 && len(deployHosts) > 0 {
		exec, err := s.taskService.ExecuteTaskWithEnv(
			pipeline.DeployTaskID,
			deployHosts,
			run.TriggeredBy,
			s.buildTaskEnv(project, pipeline, run, "deploy", nil),
		)
		if err != nil {
			stages.Deploy.Status = "failed"
			stages.Deploy.Error = err.Error()
			stages.Deploy.Summary = "部署触发失败"
			stages.CurrentStage = "done"
			run.ArtifactSummary = mustMarshalRunStages(stages)
			run.Status = "failed"
			run.Result = "deploy_failed"
			run.ErrorMessage = err.Error()
			run.Summary = "部署触发失败：" + err.Error()
			return nil
		}
		stages.Deploy.ExecutionID = exec.ID
		stages.Deploy.Status = "running"
		stages.Deploy.Summary = fmt.Sprintf("已触发部署任务 #%d", exec.ID)
		stages.Deploy.Error = ""
		stages.CurrentStage = "deploy"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.TaskExecutionID = exec.ID
		run.Status = "running"
		run.Result = ""
		run.Summary = fmt.Sprintf("已触发部署任务 #%d", exec.ID)
		return nil
	}
	stages.Deploy.Status = "skipped"
	stages.Deploy.Summary = "未配置部署阶段"
	stages.CurrentStage = "done"
	run.ArtifactSummary = mustMarshalRunStages(stages)
	run.Status = "success"
	run.Result = "success"
	run.TaskExecutionID = 0
	run.Summary = "流水线已完成（无部署阶段）"
	now := model.LocalTime(time.Now())
	run.FinishedAt = now
	return nil
}

func (s *CICDService) StartApprovedRunsByTicketID(ticketID int64) error {
	runs, err := s.runRepo.ListWaitingApprovalByTicketID(ticketID)
	if err != nil {
		return err
	}
	for _, run := range runs {
		pipeline, err := s.pipelineRepo.GetByID(run.PipelineID)
		if err != nil {
			run.Status = "failed"
			run.ErrorMessage = "流水线不存在"
			run.Summary = "审批通过后触发失败：流水线不存在"
			_ = s.saveRun(run)
			continue
		}
		stages := parseRunStages(run.ArtifactSummary)
		if stages.Deploy.ExecutionID > 0 && run.TaskExecutionID == stages.Deploy.ExecutionID {
			continue
		}
		targetHosts := parseTargetHosts(run.TargetHosts)
		if len(targetHosts) == 0 {
			targetHosts = parseTargetHosts(pipeline.TargetHosts)
		}
		stages.Approval.Status = "approved"
		stages.Approval.TicketID = ticketID
		stages.Approval.Summary = fmt.Sprintf("审批工单 #%d 已通过", ticketID)
		stages.Approval.Error = ""
		if pipeline.DeployTaskID == 0 || len(targetHosts) == 0 {
			stages.Deploy.Status = "skipped"
			stages.Deploy.Summary = "未配置部署任务或目标主机"
			stages.CurrentStage = "done"
			run.ArtifactSummary = mustMarshalRunStages(stages)
			run.Status = "success"
			run.Result = "approved"
			run.TaskExecutionID = 0
			run.Summary = "审批已通过，未配置部署任务或目标主机"
			now := model.LocalTime(time.Now())
			run.FinishedAt = now
			_ = s.saveRun(run)
			continue
		}
		project, _ := s.projectRepo.GetByID(pipeline.ProjectID)
		exec, err := s.taskService.ExecuteTaskWithEnv(
			pipeline.DeployTaskID,
			targetHosts,
			run.TriggeredBy,
			s.buildTaskEnv(project, pipeline, run, "deploy", nil),
		)
		if err != nil {
			stages.Deploy.Status = "failed"
			stages.Deploy.Error = err.Error()
			stages.Deploy.Summary = "审批通过后触发部署失败"
			stages.CurrentStage = "done"
			run.ArtifactSummary = mustMarshalRunStages(stages)
			run.Status = "failed"
			run.ErrorMessage = err.Error()
			run.Summary = "审批通过后触发部署失败：" + err.Error()
			now := model.LocalTime(time.Now())
			run.FinishedAt = now
			_ = s.saveRun(run)
			continue
		}
		stages.Deploy.ExecutionID = exec.ID
		stages.Deploy.Status = "running"
		stages.Deploy.Summary = fmt.Sprintf("审批通过后已触发部署任务 #%d", exec.ID)
		stages.Deploy.Error = ""
		stages.CurrentStage = "deploy"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.TaskExecutionID = exec.ID
		run.Status = "running"
		run.Result = ""
		run.Summary = fmt.Sprintf("审批通过后已触发部署任务 #%d", exec.ID)
		now := model.LocalTime(time.Now())
		run.FinishedAt = now
		_ = s.saveRun(run)
	}
	return nil
}

func (s *CICDService) buildTaskEnv(project *model.CICDProject, pipeline *model.CICDPipeline, run *model.CICDPipelineRun, stage string, stageVars map[string]string) map[string]string {
	env := map[string]string{
		"CICD_RUN_ID":       fmt.Sprintf("%d", run.ID),
		"CICD_RUN_NUMBER":   fmt.Sprintf("%d", run.RunNumber),
		"CICD_TRIGGER_TYPE": run.TriggerType,
		"CICD_BRANCH":       run.Branch,
		"CICD_COMMIT_ID":    run.CommitID,
	}
	if pipeline != nil {
		for key, value := range parsePipelineRuntimeConfig(pipeline.ConfigJSON, pipeline.VariablesJSON, nil).Variables {
			env[key] = value
		}
	}
	for key, value := range stageVars {
		env[key] = value
	}
	if strings.TrimSpace(stage) != "" {
		env["CICD_STAGE"] = stage
	}
	if project != nil {
		env["CICD_PROJECT_ID"] = fmt.Sprintf("%d", project.ID)
		env["CICD_PROJECT_NAME"] = project.Name
	}
	if pipeline != nil {
		env["CICD_PIPELINE_ID"] = fmt.Sprintf("%d", pipeline.ID)
		env["CICD_PIPELINE_NAME"] = pipeline.Name
		env["CICD_ENVIRONMENT"] = pipeline.Environment
	}
	if run.ApprovalTicketID > 0 {
		env["CICD_APPROVAL_TICKET_ID"] = fmt.Sprintf("%d", run.ApprovalTicketID)
	}
	if run.MetadataJSON != "" {
		var meta map[string]any
		if json.Unmarshal([]byte(run.MetadataJSON), &meta) == nil {
			if sourceRunID, ok := meta["source_run_id"]; ok {
				env["CICD_SOURCE_RUN_ID"] = fmt.Sprintf("%v", sourceRunID)
			}
		}
	}
	return env
}

func (s *CICDService) MarkRejectedRunsByTicketID(ticketID int64, reason string) error {
	runs, err := s.runRepo.ListWaitingApprovalByTicketID(ticketID)
	if err != nil {
		return err
	}
	for _, run := range runs {
		stages := parseRunStages(run.ArtifactSummary)
		stages.Approval.Status = "rejected"
		stages.Approval.TicketID = ticketID
		stages.Approval.Summary = "审批被拒绝"
		stages.Approval.Error = reason
		stages.CurrentStage = "done"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.Status = "failed"
		run.Result = "rejected"
		run.TaskExecutionID = 0
		run.ErrorMessage = reason
		if reason == "" {
			run.Summary = "审批被拒绝"
		} else {
			run.Summary = "审批被拒绝：" + reason
		}
		now := model.LocalTime(time.Now())
		run.FinishedAt = now
		if err := s.saveRun(run); err != nil {
			return err
		}
	}
	return nil
}

func (s *CICDService) syncRunExecutionStatus(run *model.CICDPipelineRun) error {
	if run == nil {
		return nil
	}
	stages := parseRunStages(run.ArtifactSummary)
	if stages.Build.ExecutionID > 0 && (stages.CurrentStage == "build" || stages.Build.Status == "running" || stages.Build.Status == "pending") {
		exec, err := s.taskExecRepo.GetByID(stages.Build.ExecutionID)
		if err == nil {
			return s.applyBuildStageStatus(run, exec)
		}
	}
	if run.TaskExecutionID == 0 {
		return nil
	}
	exec, err := s.taskExecRepo.GetByID(run.TaskExecutionID)
	if err != nil {
		return err
	}
	if stages.Build.ExecutionID > 0 && exec.ID == stages.Build.ExecutionID {
		return s.applyBuildStageStatus(run, exec)
	}
	if stages.Deploy.ExecutionID > 0 && exec.ID == stages.Deploy.ExecutionID {
		return s.applyDeployStageStatus(run, exec)
	}
	return s.applyRunStatusFromExecution(run, exec)
}

func (s *CICDService) applyBuildStageStatus(run *model.CICDPipelineRun, exec *model.TaskExecution) error {
	if run == nil || exec == nil {
		return nil
	}
	stages := parseRunStages(run.ArtifactSummary)
	if stages.CurrentStage != "" && stages.CurrentStage != "build" && stages.Build.Status == "success" {
		return nil
	}
	execStatus := strings.TrimSpace(strings.ToLower(exec.Status))
	switch execStatus {
	case "pending", "running":
		stages.Build.Status = "running"
		stages.Build.Summary = fmt.Sprintf("构建任务 #%d 正在运行", exec.ID)
		stages.CurrentStage = "build"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.Status = "running"
		run.Result = ""
		run.Summary = fmt.Sprintf("构建任务 #%d 正在运行", exec.ID)
		return s.saveRun(run)
	case "success":
		stages.Build.Status = "success"
		stages.Build.Summary = fmt.Sprintf("构建完成：成功 %d / 总数 %d", exec.SuccessCount, exec.TotalCount)
		stages.Build.Error = ""
		stages.CurrentStage = "build"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.TaskExecutionID = 0
		pipeline, err := s.pipelineRepo.GetByID(run.PipelineID)
		if err != nil {
			run.Status = "failed"
			run.Result = "build_succeeded_but_pipeline_missing"
			run.ErrorMessage = "构建成功后推进失败：流水线不存在"
			run.Summary = run.ErrorMessage
			return s.saveRun(run)
		}
		project, _ := s.projectRepo.GetByID(pipeline.ProjectID)
		deployHosts := parseTargetHosts(run.TargetHosts)
		if len(deployHosts) == 0 {
			deployHosts = parseTargetHosts(pipeline.TargetHosts)
		}
		if err := s.advanceAfterBuildSuccess(run, pipeline, project, deployHosts); err != nil {
			run.Status = "failed"
			run.Result = "build_succeeded_but_stage_advance_failed"
			run.ErrorMessage = err.Error()
			run.Summary = "构建成功后推进失败：" + err.Error()
		}
		return s.saveRun(run)
	case "failed", "partial_fail", "canceled":
		stages.Build.Status = "failed"
		stages.Build.Summary = fmt.Sprintf("构建失败：成功 %d / 失败 %d / 总数 %d", exec.SuccessCount, exec.FailCount, exec.TotalCount)
		stages.Build.Error = exec.Status
		stages.CurrentStage = "done"
		run.ArtifactSummary = mustMarshalRunStages(stages)
		run.Status = "failed"
		run.Result = "build_failed"
		run.TaskExecutionID = 0
		run.Summary = fmt.Sprintf("构建失败：成功 %d / 失败 %d / 总数 %d", exec.SuccessCount, exec.FailCount, exec.TotalCount)
		if exec.FinishedAt != nil {
			run.FinishedAt = *exec.FinishedAt
		}
		return s.saveRun(run)
	default:
		return nil
	}
}

func (s *CICDService) applyDeployStageStatus(run *model.CICDPipelineRun, exec *model.TaskExecution) error {
	if run == nil || exec == nil {
		return nil
	}
	stages := parseRunStages(run.ArtifactSummary)
	if err := s.applyRunStatusFromExecution(run, exec); err != nil {
		return err
	}
	switch strings.TrimSpace(strings.ToLower(exec.Status)) {
	case "pending", "running":
		stages.Deploy.Status = "running"
		stages.Deploy.Summary = fmt.Sprintf("部署任务 #%d 正在运行", exec.ID)
		stages.Deploy.Error = ""
		stages.CurrentStage = "deploy"
	case "success":
		stages.Deploy.Status = "success"
		stages.Deploy.Summary = fmt.Sprintf("部署完成：成功 %d / 总数 %d", exec.SuccessCount, exec.TotalCount)
		stages.Deploy.Error = ""
		stages.CurrentStage = "done"
	case "failed", "partial_fail", "canceled":
		stages.Deploy.Status = "failed"
		stages.Deploy.Summary = fmt.Sprintf("部署失败：成功 %d / 失败 %d / 总数 %d", exec.SuccessCount, exec.FailCount, exec.TotalCount)
		stages.Deploy.Error = exec.Status
		stages.CurrentStage = "done"
	}
	run.ArtifactSummary = mustMarshalRunStages(stages)
	return s.saveRun(run)
}

func (s *CICDService) applyRunStatusFromExecution(run *model.CICDPipelineRun, exec *model.TaskExecution) error {
	if run == nil || exec == nil {
		return nil
	}
	originalStatus := run.Status
	originalResult := run.Result
	originalSummary := run.Summary
	originalFinishedAt := run.FinishedAt

	execStatus := strings.TrimSpace(strings.ToLower(exec.Status))
	switch execStatus {
	case "pending", "running":
		run.Status = "running"
		run.Result = ""
		if run.Summary == "" || strings.Contains(run.Summary, "已触发任务执行") {
			run.Summary = fmt.Sprintf("任务执行 #%d 正在运行", exec.ID)
		}
	case "success":
		run.Status = "success"
		run.Result = "success"
		run.Summary = fmt.Sprintf("部署完成：成功 %d / 总数 %d", exec.SuccessCount, exec.TotalCount)
	case "failed", "partial_fail":
		run.Status = "failed"
		run.Result = execStatus
		run.Summary = fmt.Sprintf("部署失败：成功 %d / 失败 %d / 总数 %d", exec.SuccessCount, exec.FailCount, exec.TotalCount)
	case "canceled":
		run.Status = "canceled"
		run.Result = "canceled"
		if run.Summary == "" {
			run.Summary = "部署任务已取消"
		}
	default:
		if exec.FinishedAt != nil {
			if exec.FailCount > 0 {
				run.Status = "failed"
				run.Result = execStatus
				run.Summary = fmt.Sprintf("部署失败：成功 %d / 失败 %d / 总数 %d", exec.SuccessCount, exec.FailCount, exec.TotalCount)
			} else if exec.SuccessCount > 0 && exec.SuccessCount == exec.TotalCount {
				run.Status = "success"
				run.Result = "success"
				run.Summary = fmt.Sprintf("部署完成：成功 %d / 总数 %d", exec.SuccessCount, exec.TotalCount)
			}
		}
	}

	if exec.FinishedAt != nil {
		run.FinishedAt = *exec.FinishedAt
		startTime := time.Time(run.StartedAt)
		endTime := time.Time(*exec.FinishedAt)
		if !startTime.IsZero() && endTime.After(startTime) {
			run.DurationSeconds = int(endTime.Sub(startTime).Seconds())
		}
	}

	if run.Status != originalStatus || run.Result != originalResult || run.Summary != originalSummary || run.FinishedAt != originalFinishedAt {
		return s.saveRun(run)
	}
	return nil
}

func (s *CICDService) saveRun(run *model.CICDPipelineRun) error {
	previousStatus := ""
	if run != nil && run.ID > 0 {
		if existing, err := s.runRepo.GetByID(run.ID); err == nil && existing != nil {
			previousStatus = existing.Status
		}
	}
	if err := s.runRepo.Update(run); err != nil {
		return err
	}
	if shouldNotifyPipelineStatusTransition(previousStatus, run.Status) {
		s.notifyPipelineRun(run)
	}
	return nil
}

func shouldNotifyPipelineStatusTransition(previousStatus, currentStatus string) bool {
	if previousStatus == currentStatus {
		return false
	}
	switch currentStatus {
	case "success", "failed", "canceled":
		return true
	default:
		return false
	}
}

func (s *CICDService) notifyPipelineRun(run *model.CICDPipelineRun) {
	if run == nil {
		return
	}
	eventType := ""
	level := "info"
	switch run.Status {
	case "success":
		eventType = "pipeline_succeeded"
		level = "info"
	case "failed":
		eventType = "pipeline_failed"
		level = "error"
	case "canceled":
		eventType = "pipeline_failed"
		level = "warning"
	default:
		return
	}
	title := fmt.Sprintf("流水线 %s #%d %s", firstNonEmpty(run.PipelineName, fmt.Sprintf("%d", run.PipelineID)), run.RunNumber, run.Status)
	content := fmt.Sprintf("项目 %s %s 分支 %s，结果 %s", run.ProjectName, run.PipelineName, run.Branch, run.Result)
	userIDs, channels, notifyConfig := s.resolvePipelineNotifyTargets(run)
	_, _ = s.notifySvc.Publish(NotificationPublishRequest{
		EventType:    eventType,
		BizType:      "cicd_pipeline",
		BizID:        run.ID,
		Title:        title,
		Content:      content,
		Level:        level,
		NotifyConfig: notifyConfig,
		Payload: map[string]interface{}{
			"pipeline_id":   run.PipelineID,
			"pipeline_name": run.PipelineName,
			"run_id":        run.ID,
			"status":        run.Status,
			"result":        run.Result,
			"branch":        run.Branch,
		},
		UserIDs:  userIDs,
		Channels: channels,
	})
}

func (s *CICDService) resolvePipelineNotifyTargets(run *model.CICDPipelineRun) ([]int64, []string, map[string]WebhookTarget) {
	if run == nil || run.PipelineID <= 0 {
		return nil, nil, nil
	}
	pipeline, err := s.pipelineRepo.GetByID(run.PipelineID)
	if err != nil || pipeline == nil {
		return nil, nil, nil
	}
	cfg := parsePipelineRuntimeConfig(pipeline.ConfigJSON, pipeline.VariablesJSON, nil)
	var userIDs []int64
	if run.ProjectID > 0 {
		if project, err := s.projectRepo.GetByID(run.ProjectID); err == nil && project.OwnerID > 0 {
			userIDs = append(userIDs, project.OwnerID)
		}
	}
	if run.TriggeredBy > 0 {
		userIDs = append(userIDs, run.TriggeredBy)
	}
	// 从 config_json 中提取 notify_config
	notifyConfig := ParseNotifyConfig(extractStringFromJSON(pipeline.ConfigJSON, "notify_config"))
	return dedupeRunNotifyUserIDs(userIDs), cfg.NotifyChannels, notifyConfig
}

func (s *CICDService) validatePipelineRefs(item *model.CICDPipeline) error {
	if item.BuildTaskID > 0 {
		if _, err := s.taskRepo.GetTask(item.BuildTaskID); err != nil {
			return errors.New("构建任务不存在")
		}
	}
	if item.DeployTaskID > 0 {
		if _, err := s.taskRepo.GetTask(item.DeployTaskID); err != nil {
			return errors.New("部署任务不存在")
		}
	}
	if item.RequestTemplateID > 0 {
		if _, err := s.requestRepo.GetByID(item.RequestTemplateID); err != nil {
			return errors.New("审批模板不存在")
		}
	}
	if !validJSONArray(item.TargetHosts) {
		return errors.New("目标主机格式错误")
	}
	if strings.TrimSpace(item.ConfigJSON) != "" {
		var cfg map[string]any
		if err := json.Unmarshal([]byte(item.ConfigJSON), &cfg); err != nil {
			return errors.New("配置 JSON 格式错误")
		}
	}
	if strings.TrimSpace(item.VariablesJSON) != "" {
		var vars map[string]any
		if err := json.Unmarshal([]byte(item.VariablesJSON), &vars); err != nil {
			return errors.New("变量 JSON 格式错误")
		}
	}
	return nil
}

func validJSONArray(raw string) bool {
	if raw == "" {
		return true
	}
	var items []string
	return json.Unmarshal([]byte(raw), &items) == nil
}

func parseTargetHosts(raw string) []string {
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func parsePipelineRuntimeConfig(configJSON, variablesJSON string, fallbackBuildHosts []string) pipelineRuntimeConfig {
	cfg := pipelineRuntimeConfig{
		BuildHosts: normalizeStringSlice(fallbackBuildHosts),
		Variables:  make(map[string]string),
		NotifyChannels: []string{},
	}
	var pipelineVars map[string]any
	if strings.TrimSpace(variablesJSON) != "" && json.Unmarshal([]byte(variablesJSON), &pipelineVars) == nil {
		for key, value := range pipelineVars {
			cfg.Variables[key] = fmt.Sprintf("%v", value)
		}
	}
	var config map[string]any
	if strings.TrimSpace(configJSON) != "" && json.Unmarshal([]byte(configJSON), &config) == nil {
		if enabled, ok := config["webhook_enabled"].(bool); ok {
			cfg.WebhookEnabled = enabled
		}
		if secret, ok := config["webhook_secret"].(string); ok {
			cfg.WebhookSecret = strings.TrimSpace(secret)
		}
		if rawHosts, ok := config["build_hosts"].([]any); ok {
			hosts := make([]string, 0, len(rawHosts))
			for _, item := range rawHosts {
				hosts = append(hosts, fmt.Sprintf("%v", item))
			}
			hosts = normalizeStringSlice(hosts)
			if len(hosts) > 0 {
				cfg.BuildHosts = hosts
			}
		}
		if rawVars, ok := config["variables"].(map[string]any); ok {
			for key, value := range rawVars {
				cfg.Variables[key] = fmt.Sprintf("%v", value)
			}
		}
		if rawChannels, ok := config["notify_channels"].([]any); ok {
			channels := make([]string, 0, len(rawChannels))
			for _, item := range rawChannels {
				value := strings.TrimSpace(fmt.Sprintf("%v", item))
				if value == "" {
					continue
				}
				channels = append(channels, value)
			}
			cfg.NotifyChannels = normalizeStringSlice(channels)
		}
	}
	return cfg
}

func normalizeStringSlice(items []string) []string {
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func extractStringFromJSON(raw, key string) string {
	var m map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &m) != nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	return string(v)
}

func dedupeRunNotifyUserIDs(items []int64) []int64 {
	result := make([]int64, 0, len(items))
	seen := make(map[int64]struct{}, len(items))
	for _, item := range items {
		if item <= 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func parseRunStages(raw string) cicdRunStageSummary {
	stages := cicdRunStageSummary{
		Variables: make(map[string]string),
	}
	if strings.TrimSpace(raw) == "" {
		return stages
	}
	if err := json.Unmarshal([]byte(raw), &stages); err != nil {
		return cicdRunStageSummary{Variables: make(map[string]string)}
	}
	if stages.Variables == nil {
		stages.Variables = make(map[string]string)
	}
	return stages
}

func mustMarshalRunStages(stages cicdRunStageSummary) string {
	data, err := json.Marshal(stages)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func mustMarshalMapString(values map[string]string) string {
	if values == nil {
		return "{}"
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func (s *CICDService) fillProjectOwners(items []*model.CICDProject) {
	userIDs := make([]int64, 0)
	seen := make(map[int64]struct{})
	for _, item := range items {
		if item.OwnerID > 0 {
			if _, ok := seen[item.OwnerID]; !ok {
				seen[item.OwnerID] = struct{}{}
				userIDs = append(userIDs, item.OwnerID)
			}
		}
	}
	nameMap := s.userRepo.GetNamesByIDs(userIDs)
	for _, item := range items {
		item.OwnerName = nameMap[item.OwnerID]
	}
}

func (s *CICDService) fillPipelines(items []*model.CICDPipeline) {
	projectIDs := make([]int64, 0)
	taskIDs := make([]int64, 0)
	requestIDs := make([]int64, 0)
	projectSeen := make(map[int64]struct{})
	taskSeen := make(map[int64]struct{})
	requestSeen := make(map[int64]struct{})
	for _, item := range items {
		if item.ProjectID > 0 {
			if _, ok := projectSeen[item.ProjectID]; !ok {
				projectSeen[item.ProjectID] = struct{}{}
				projectIDs = append(projectIDs, item.ProjectID)
			}
		}
		for _, taskID := range []int64{item.BuildTaskID, item.DeployTaskID} {
			if taskID > 0 {
				if _, ok := taskSeen[taskID]; !ok {
					taskSeen[taskID] = struct{}{}
					taskIDs = append(taskIDs, taskID)
				}
			}
		}
		if item.RequestTemplateID > 0 {
			if _, ok := requestSeen[item.RequestTemplateID]; !ok {
				requestSeen[item.RequestTemplateID] = struct{}{}
				requestIDs = append(requestIDs, item.RequestTemplateID)
			}
		}
	}
	projectMap := make(map[int64]*model.CICDProject)
	for _, id := range projectIDs {
		if item, err := s.projectRepo.GetByID(id); err == nil {
			projectMap[id] = item
		}
	}
	taskMap := make(map[int64]*model.Task)
	for _, id := range taskIDs {
		if item, err := s.taskRepo.GetTask(id); err == nil {
			taskMap[id] = item
		}
	}
	requestMap, _ := s.requestRepo.GetByIDs(requestIDs)
	for _, item := range items {
		if project, ok := projectMap[item.ProjectID]; ok {
			item.ProjectName = project.Name
		}
		if task, ok := taskMap[item.BuildTaskID]; ok {
			item.BuildTaskName = task.Name
		}
		if task, ok := taskMap[item.DeployTaskID]; ok {
			item.DeployTaskName = task.Name
		}
		if tpl, ok := requestMap[item.RequestTemplateID]; ok {
			item.RequestTemplateName = tpl.Name
		}
	}
}

func (s *CICDService) fillRuns(items []*model.CICDPipelineRun) {
	projectIDs := make([]int64, 0)
	pipelineIDs := make([]int64, 0)
	userIDs := make([]int64, 0)
	projectSeen := make(map[int64]struct{})
	pipelineSeen := make(map[int64]struct{})
	userSeen := make(map[int64]struct{})
	for _, item := range items {
		if item.ProjectID > 0 {
			if _, ok := projectSeen[item.ProjectID]; !ok {
				projectSeen[item.ProjectID] = struct{}{}
				projectIDs = append(projectIDs, item.ProjectID)
			}
		}
		if item.PipelineID > 0 {
			if _, ok := pipelineSeen[item.PipelineID]; !ok {
				pipelineSeen[item.PipelineID] = struct{}{}
				pipelineIDs = append(pipelineIDs, item.PipelineID)
			}
		}
		if item.TriggeredBy > 0 {
			if _, ok := userSeen[item.TriggeredBy]; !ok {
				userSeen[item.TriggeredBy] = struct{}{}
				userIDs = append(userIDs, item.TriggeredBy)
			}
		}
	}
	projectMap := make(map[int64]*model.CICDProject)
	for _, id := range projectIDs {
		if item, err := s.projectRepo.GetByID(id); err == nil {
			projectMap[id] = item
		}
	}
	pipelineMap := make(map[int64]*model.CICDPipeline)
	for _, id := range pipelineIDs {
		if item, err := s.pipelineRepo.GetByID(id); err == nil {
			pipelineMap[id] = item
		}
	}
	nameMap := s.userRepo.GetNamesByIDs(userIDs)
	for _, item := range items {
		if project, ok := projectMap[item.ProjectID]; ok {
			item.ProjectName = project.Name
		}
		if pipeline, ok := pipelineMap[item.PipelineID]; ok {
			item.PipelineName = pipeline.Name
			item.PipelineCode = pipeline.Code
		}
		item.TriggeredByName = nameMap[item.TriggeredBy]
	}
}

func (s *CICDService) getUserName(id int64) string {
	if id == 0 {
		return ""
	}
	if user, err := s.userRepo.GetByID(id); err == nil {
		if user.RealName != "" {
			return user.RealName
		}
		return user.Username
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
