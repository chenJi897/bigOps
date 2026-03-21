// backend/internal/model/asset_change.go
package model

// AssetChange 资产变更历史模型，对应 asset_changes 表。
type AssetChange struct {
	ID           int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	AssetID      int64     `gorm:"index;not null" json:"asset_id"`
	Field        string    `gorm:"size:50;not null" json:"field"`
	OldValue     string    `gorm:"type:text" json:"old_value"`
	NewValue     string    `gorm:"type:text" json:"new_value"`
	ChangeType   string    `gorm:"size:20;not null" json:"change_type"` // create/update/sync/delete
	OperatorID   int64     `gorm:"default:0" json:"operator_id"`
	OperatorName string    `gorm:"size:50" json:"operator_name"`
	CreatedAt    LocalTime `json:"created_at" swaggertype:"string" example:"2024-01-01 00:00:00"`
}

func (AssetChange) TableName() string {
	return "asset_changes"
}
