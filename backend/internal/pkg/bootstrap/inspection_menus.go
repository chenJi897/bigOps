// Package bootstrap 提供启动时幂等数据修复，避免仅依赖手工执行 SQL 迁移。
package bootstrap

import (
	"errors"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EnsureInspectionMenus 幂等写入「巡检中心」目录与页面菜单（/inspection/center）。
// 目标：让前端动态路由能加载 InspectionCenter 组件，并给 admin 角色默认赋权可见。
func EnsureInspectionMenus(db *gorm.DB) error {
	// 目录：inspection_dir（顶层目录）
	dirDesired := model.Menu{
		ParentID:  0,
		Name:      "inspection_dir",
		Title:     "巡检中心",
		Icon:      "Document",
		Path:      "/inspection",
		Component: "",
		APIPath:   "",
		APIMethod: "",
		Type:      1,
		Sort:      80,
		Visible:   1,
		Status:    1,
	}

	var dir model.Menu
	dirErr := db.Unscoped().Where("name = ?", dirDesired.Name).First(&dir).Error
	if errors.Is(dirErr, gorm.ErrRecordNotFound) {
		if err := db.Create(&dirDesired).Error; err != nil {
			return err
		}
		dir = dirDesired
		logger.Get().Info("bootstrap: created menu inspection_dir")
	} else if dirErr != nil {
		return dirErr
	} else {
		dir.ParentID = dirDesired.ParentID
		dir.Title = dirDesired.Title
		dir.Icon = dirDesired.Icon
		dir.Path = dirDesired.Path
		dir.Component = dirDesired.Component
		dir.Type = dirDesired.Type
		dir.Sort = dirDesired.Sort
		dir.Visible = dirDesired.Visible
		dir.Status = dirDesired.Status
		dir.DeletedAt = gorm.DeletedAt{}
		if err := db.Unscoped().Save(&dir).Error; err != nil {
			return err
		}
		logger.Get().Info("bootstrap: upserted menu inspection_dir")
	}

	// 页面：inspection_center
	pageDesired := model.Menu{
		ParentID:  dir.ID,
		Name:      "inspection_center",
		Title:     "巡检中心",
		Icon:      "DocumentChecked",
		Path:      "/inspection/center",
		Component: "InspectionCenter",
		APIPath:   "/api/v1/inspection/*",
		APIMethod: "*",
		Type:      2,
		Sort:      1,
		Visible:   1,
		Status:    1,
	}

	var page model.Menu
	pageErr := db.Unscoped().Where("name = ?", pageDesired.Name).First(&page).Error
	if errors.Is(pageErr, gorm.ErrRecordNotFound) {
		if err := db.Create(&pageDesired).Error; err != nil {
			return err
		}
		page = pageDesired
		logger.Get().Info("bootstrap: created menu inspection_center")
	} else if pageErr != nil {
		return pageErr
	} else {
		page.ParentID = pageDesired.ParentID
		page.Title = pageDesired.Title
		page.Icon = pageDesired.Icon
		page.Path = pageDesired.Path
		page.Component = pageDesired.Component
		page.APIPath = pageDesired.APIPath
		page.APIMethod = pageDesired.APIMethod
		page.Type = pageDesired.Type
		page.Sort = pageDesired.Sort
		page.Visible = pageDesired.Visible
		page.Status = pageDesired.Status
		page.DeletedAt = gorm.DeletedAt{}
		if err := db.Unscoped().Save(&page).Error; err != nil {
			return err
		}
		logger.Get().Info("bootstrap: upserted menu inspection_center")
	}

	// 默认赋权给 admin 角色
	var admin model.Role
	if err := db.Where("name = ? AND deleted_at IS NULL", "admin").First(&admin).Error; err != nil {
		return nil
	}

	// role_menus：目录与页面都给（避免前端树过滤）
	if err := db.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", admin.ID, dir.ID).Error; err != nil {
		logger.Get().Warn("bootstrap: role_menus admin inspection_dir", zap.Error(err))
	}
	if err := db.Exec("INSERT IGNORE INTO role_menus (role_id, menu_id) VALUES (?, ?)", admin.ID, page.ID).Error; err != nil {
		logger.Get().Warn("bootstrap: role_menus admin inspection_center", zap.Error(err))
	}

	return nil
}

