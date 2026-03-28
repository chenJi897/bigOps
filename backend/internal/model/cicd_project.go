package model

import "gorm.io/gorm"

// CICDProject 表示一个 CI/CD 项目，包含仓库信息与归属团队。
type CICDProject struct {
	ID            int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name          string         `gorm:"size:120;not null;uniqueIndex" json:"name"`
	Code          string         `gorm:"size:60;not null;uniqueIndex" json:"code"`
	Description   string         `gorm:"size:500" json:"description"`
	RepoProvider  string         `gorm:"size:30;default:git" json:"repo_provider"`
	RepoURL       string         `gorm:"size:255;not null" json:"repo_url"`
	DefaultBranch string         `gorm:"size:100;default:main" json:"default_branch"`
	Visibility    string         `gorm:"size:30;default:private" json:"visibility"` // private/public
	TeamID        int64          `gorm:"index" json:"team_id"`
	TeamName      string         `gorm:"-" json:"team_name,omitempty"`
	OwnerID       int64          `gorm:"index" json:"owner_id"`
	OwnerName     string         `gorm:"-" json:"owner_name,omitempty"`
	Status        int8           `gorm:"default:1;index" json:"status"`
	ConfigJSON    string         `gorm:"type:json" json:"config_json"`
	CreatedAt     LocalTime      `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	UpdatedAt     LocalTime      `json:"updated_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (CICDProject) TableName() string {
	return "cicd_projects"
}

func (p *CICDProject) BeforeSave(tx *gorm.DB) error {
	if p.ConfigJSON == "" {
		p.ConfigJSON = "{}"
	}
	return nil
}
