package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"strings"
	"unicode"

	"gorm.io/gorm"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type RequestTemplateService struct {
	repo       *repository.RequestTemplateRepository
	policyRepo *repository.ApprovalPolicyRepository
	typeRepo   *repository.TicketTypeRepository
}

func NewRequestTemplateService() *RequestTemplateService {
	return &RequestTemplateService{
		repo:       repository.NewRequestTemplateRepository(),
		policyRepo: repository.NewApprovalPolicyRepository(),
		typeRepo:   repository.NewTicketTypeRepository(),
	}
}

func (s *RequestTemplateService) Create(item *model.RequestTemplate) error {
	if item.Name == "" {
		return errors.New("请求模板名称不能为空")
	}
	if err := validateNodesJSON(item.NodesJSON); err != nil {
		return err
	}
	if strings.TrimSpace(item.Code) == "" {
		code, err := s.generateCode(item.Name, 0)
		if err != nil {
			return err
		}
		item.Code = code
	}
	if item.TypeID == 0 {
		// TypeID 为 0 时不强制关联工单类型，使用 category 作为分类
	} else {
		if _, err := s.typeRepo.GetByID(item.TypeID); err != nil {
			return errors.New("工单类型不存在")
		}
	}
	if item.Category == "" {
		item.Category = "other"
	}
	if item.TicketKind == "" {
		item.TicketKind = "request"
	}
	if item.ApprovalPolicyID > 0 {
		if _, err := s.policyRepo.GetByID(item.ApprovalPolicyID); err != nil {
			return errors.New("审批策略不存在")
		}
	}
	if item.TypeID > 0 {
		if _, err := s.typeRepo.GetByID(item.TypeID); err != nil {
			return errors.New("工单类型不存在")
		}
	}
	if _, err := s.repo.GetByName(item.Name); err == nil {
		return errors.New("请求模板名称已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	if _, err := s.repo.GetByCode(item.Code); err == nil {
		return errors.New("请求模板编码已存在")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询失败: %w", err)
	}
	return s.repo.Create(item)
}

func (s *RequestTemplateService) Update(id int64, item *model.RequestTemplate) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("请求模板不存在")
	}
	if item.Name == "" {
		return errors.New("请求模板名称不能为空")
	}
	if err := validateNodesJSON(item.NodesJSON); err != nil {
		return err
	}
	if strings.TrimSpace(item.Code) == "" {
		item.Code = existing.Code
		if item.Code == "" {
			code, err := s.generateCode(item.Name, id)
			if err != nil {
				return err
			}
			item.Code = code
		}
	}
	if item.TypeID == 0 {
		item.TypeID = existing.TypeID
	}
	if item.ApprovalPolicyID > 0 {
		if _, err := s.policyRepo.GetByID(item.ApprovalPolicyID); err != nil {
			return errors.New("审批策略不存在")
		}
	}
	if item.TypeID > 0 {
		if _, err := s.typeRepo.GetByID(item.TypeID); err != nil {
			return errors.New("工单类型不存在")
		}
	}
	if item.Name != existing.Name {
		if dup, err := s.repo.GetByName(item.Name); err == nil && dup.ID != id {
			return errors.New("请求模板名称已存在")
		}
	}
	if item.Code != existing.Code {
		if dup, err := s.repo.GetByCode(item.Code); err == nil && dup.ID != id {
			return errors.New("请求模板编码已存在")
		}
	}
	existing.Name = item.Name
	existing.Code = item.Code
	existing.Category = item.Category
	existing.ProjectName = item.ProjectName
	existing.EnvironmentName = item.EnvironmentName
	existing.Description = item.Description
	existing.Icon = item.Icon
	existing.TypeID = item.TypeID
	existing.FormSchema = item.FormSchema
	existing.ApprovalPolicyID = item.ApprovalPolicyID
	existing.NodesJSON = item.NodesJSON
	existing.ExecutionTemplate = item.ExecutionTemplate
	existing.TicketKind = item.TicketKind
	existing.Priority = item.Priority
	existing.HandleDeptID = item.HandleDeptID
	existing.AutoAssignRule = item.AutoAssignRule
	existing.DefaultAssignee = item.DefaultAssignee
	existing.AutoCreateOrder = item.AutoCreateOrder
	existing.NotifyApplicant = item.NotifyApplicant
	existing.Sort = item.Sort
	existing.Status = item.Status
	return s.repo.Update(existing)
}

func (s *RequestTemplateService) Delete(id int64) error {
	if _, err := s.repo.GetByID(id); err != nil {
		return errors.New("请求模板不存在")
	}
	return s.repo.Delete(id)
}

func (s *RequestTemplateService) GetByID(id int64) (*model.RequestTemplate, error) {
	item, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	s.fillExtra(item)
	return item, nil
}

func (s *RequestTemplateService) List(enabledOnly bool) ([]*model.RequestTemplate, error) {
	items, err := s.repo.List(enabledOnly)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		s.fillExtra(item)
	}
	return items, nil
}

func (s *RequestTemplateService) fillExtra(item *model.RequestTemplate) {
	if item.ApprovalPolicyID > 0 {
		if policy, err := s.policyRepo.GetByID(item.ApprovalPolicyID); err == nil {
			item.ApprovalPolicyName = policy.Name
		}
	}
	if item.TypeID > 0 {
		if tt, err := s.typeRepo.GetByID(item.TypeID); err == nil {
			item.TypeName = tt.Name
		}
	}
}

func (s *RequestTemplateService) generateCode(name string, currentID int64) (string, error) {
	base := buildTemplateCodeBase(name)
	code := base
	for i := 1; i <= 1000; i++ {
		existing, err := s.repo.GetByCode(code)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return code, nil
		}
		if err != nil {
			return "", fmt.Errorf("生成请求模板编码失败: %w", err)
		}
		if existing.ID == currentID {
			return code, nil
		}
		code = fmt.Sprintf("%s-%d", trimCodeBase(base, 44), i+1)
	}
	return "", errors.New("生成请求模板编码失败")
}

func buildTemplateCodeBase(name string) string {
	raw := strings.TrimSpace(strings.ToLower(name))
	var builder strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			builder.WriteRune(r)
			lastDash = false
		case !lastDash && builder.Len() > 0:
			builder.WriteByte('-')
			lastDash = true
		}
	}
	code := strings.Trim(builder.String(), "-")
	if code == "" {
		code = fmt.Sprintf("template-%08x", crc32.ChecksumIEEE([]byte(name)))
	}
	return trimCodeBase(code, 50)
}

func trimCodeBase(code string, limit int) string {
	if len(code) <= limit {
		return code
	}
	return strings.Trim(code[:limit], "-")
}

func (s *RequestTemplateService) pickDefaultTicketType() (*model.TicketType, error) {
	items, err := s.typeRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("查询工单类型失败: %w", err)
	}
	if len(items) == 0 {
		return nil, errors.New("当前还没有可用的底层工单类型，请先创建一个工单类型")
	}
	return items[0], nil
}

// validateNodesJSON 校验节点配置至少包含 2 个节点。
func validateNodesJSON(nodesJSON string) error {
	if nodesJSON == "" || nodesJSON == "[]" || nodesJSON == "null" {
		return errors.New("节点配置不能为空，至少需要 2 个节点")
	}
	var nodes []json.RawMessage
	if err := json.Unmarshal([]byte(nodesJSON), &nodes); err != nil {
		return errors.New("节点配置格式错误")
	}
	if len(nodes) < 2 {
		return fmt.Errorf("至少需要 2 个节点，当前只有 %d 个", len(nodes))
	}
	return nil
}
