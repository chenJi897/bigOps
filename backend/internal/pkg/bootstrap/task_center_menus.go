// Package bootstrap 提供启动时幂等数据修复，避免仅依赖手工执行 SQL 迁移。
package bootstrap

import (
	"errors"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EnsureTaskExecutionsMenu 幂等写入任务中心「执行记录」菜单（/task/executions）。
// 无 task_dir 时静默跳过。
func EnsureTaskExecutionsMenu(db *gorm.DB) error {
	var dir model.Menu
	if err := db.Where("name = ? AND deleted_at IS NULL", "task_dir").First(&dir).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	desired := model.Menu{
		ParentID:  dir.ID,
		Name:      "task_executions",
		Title:     "执行记录",
		Icon:      "Document",
		Path:      "/task/executions",
		Component: "TaskExecutions",
		APIPath:   "/api/v1/task-executions*",
		APIMethod: "*",
		Type:      2,
		Sort:      2,
		Visible:   1,
		Status:    1,
	}

	var existing model.Menu
	err := db.Unscoped().Where("name = ?", "task_executions").First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := db.Create(&desired).Error; err != nil {
			return err
		}
		logger.Get().Info("bootstrap: created menu task_executions")
	} else if err != nil {
		return err
	} else {
		existing.ParentID = desired.ParentID
		existing.Title = desired.Title
		existing.Icon = desired.Icon
		existing.Path = desired.Path
		existing.Component = desired.Component
		existing.APIPath = desired.APIPath
		existing.APIMethod = desired.APIMethod
		existing.Type = desired.Type
		existing.Sort = desired.Sort
		existing.Visible = desired.Visible
		existing.Status = desired.Status
		existing.DeletedAt = gorm.DeletedAt{}
		if err := db.Unscoped().Save(&existing).Error; err != nil {
			return err
		}
		logger.Get().Info("bootstrap: upserted menu task_executions")
	}

	order := map[string]int{
		"task_list":       1,
		"task_executions": 2,
		"task_create":     3,
		"task_execution":  4,
		"agent_list":      5,
	}
	for name, sort := range order {
		if err := db.Model(&model.Menu{}).Where("name = ? AND deleted_at IS NULL", name).Update("sort", sort).Error; err != nil {
			logger.Get().Warn("bootstrap: reorder menu sort", zap.String("name", name), zap.Error(err))
		}
	}

	var admin model.Role
	if err := db.Where("name = ? AND deleted_at IS NULL", "admin").First(&admin).Error; err != nil {
		return nil
	}
	var m model.Menu
	if err := db.Where("name = ?", "task_executions").First(&m).Error; err != nil {
		return nil
	}
	if err := db.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", admin.ID, m.ID).Error; err != nil {
		logger.Get().Warn("bootstrap: role_menus admin task_executions", zap.Error(err))
	}
	return nil
}
