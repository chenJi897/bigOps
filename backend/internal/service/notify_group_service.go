package service

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/repository"
)

type NotifyGroupService struct {
	repo *repository.NotifyGroupRepository
}

func NewNotifyGroupService() *NotifyGroupService {
	return &NotifyGroupService{repo: repository.NewNotifyGroupRepository()}
}

func (s *NotifyGroupService) List(page, size int, keyword string) ([]*model.NotifyGroup, int64, error) {
	return s.repo.List(page, size, keyword)
}

func (s *NotifyGroupService) ListAll() ([]*model.NotifyGroup, error) {
	return s.repo.ListAll()
}

func (s *NotifyGroupService) GetByID(id int64) (*model.NotifyGroup, error) {
	return s.repo.GetByID(id)
}

func (s *NotifyGroupService) Create(item *model.NotifyGroup) error {
	if strings.TrimSpace(item.Name) == "" {
		return errors.New("发送组名称不能为空")
	}
	return s.repo.Create(item)
}

func (s *NotifyGroupService) Update(id int64, item *model.NotifyGroup) error {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("发送组不存在")
	}
	existing.Name = item.Name
	existing.Description = item.Description
	existing.WebhooksJSON = item.WebhooksJSON
	existing.NotifyUserIDs = item.NotifyUserIDs
	existing.RepeatEnabled = item.RepeatEnabled
	existing.RepeatIntervalSeconds = item.RepeatIntervalSeconds
	existing.SendResolved = item.SendResolved
	existing.EscalationEnabled = item.EscalationEnabled
	existing.EscalationMinutes = item.EscalationMinutes
	existing.EscalationUserIDs = item.EscalationUserIDs
	existing.EscalationWebhooksJSON = item.EscalationWebhooksJSON
	existing.Status = item.Status
	return s.repo.Update(existing)
}

func (s *NotifyGroupService) Delete(id int64) error {
	return s.repo.Delete(id)
}

// ResolveWebhookTargets 将发送组的 webhooks_json 解析为 NotifyConfig map。
func ResolveGroupWebhookTargets(webhooksJSON string) map[string]WebhookTarget {
	var webhooks []model.GroupWebhook
	if err := json.Unmarshal([]byte(webhooksJSON), &webhooks); err != nil {
		return nil
	}
	result := make(map[string]WebhookTarget)
	for i, wh := range webhooks {
		if strings.TrimSpace(wh.WebhookURL) == "" {
			continue
		}
		// 用 channel_type + index 作为 key 避免同类型多个群覆盖
		key := wh.ChannelType
		if i > 0 {
			// 检查是否已有同类型
			for j := 0; j < i; j++ {
				if webhooks[j].ChannelType == wh.ChannelType {
					key = wh.ChannelType + "_" + wh.Label
					break
				}
			}
		}
		result[key] = WebhookTarget{
			WebhookURL: wh.WebhookURL,
			Secret:     wh.Secret,
		}
	}
	return result
}
