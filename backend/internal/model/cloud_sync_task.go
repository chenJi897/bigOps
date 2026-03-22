package model

// CloudSyncTask 云同步任务记录，对应 cloud_sync_tasks 表。
// 每次同步（无论手动还是定时）都会创建一条记录。
type CloudSyncTask struct {
	ID             int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	CloudAccountID int64      `gorm:"index;not null" json:"cloud_account_id"`
	AccountName    string     `gorm:"size:100" json:"account_name"`         // 冗余，方便查询展示
	Provider       string     `gorm:"size:50" json:"provider"`              // aliyun/tencent/aws
	TriggerType    string     `gorm:"size:20;not null" json:"trigger_type"` // manual / schedule
	Status         string     `gorm:"size:20;not null;index" json:"status"` // running / success / failed
	StartedAt      LocalTime  `json:"started_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	FinishedAt     *LocalTime `json:"finished_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
	DurationMs     int64      `json:"duration_ms"`                         // 耗时毫秒
	TotalCount     int        `json:"total_count"`                         // 云端返回的实例总数
	CreatedCount   int        `json:"created_count"`                       // 新增数
	UpdatedCount   int        `json:"updated_count"`                       // 更新数
	UnchangedCount int        `json:"unchanged_count"`                     // 无变化数
	OfflineCount   int        `json:"offline_count"`                       // 下线数（云端消失）
	ErrorMessage   string     `gorm:"type:text" json:"error_message"`      // 失败原因
	OperatorID     int64      `gorm:"default:0" json:"operator_id"`        // 手动触发时的操作人
	OperatorName   string     `gorm:"size:50" json:"operator_name"`        // 操作人用户名
	Regions        string     `gorm:"size:500" json:"regions"`             // 本次同步的 region 列表
	CreatedAt      LocalTime  `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (CloudSyncTask) TableName() string {
	return "cloud_sync_tasks"
}
