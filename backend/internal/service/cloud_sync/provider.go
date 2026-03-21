package cloudsync

import "github.com/bigops/platform/internal/model"

// SyncResult 同步结果。
type SyncResult struct {
	Created int
	Updated int
	Total   int
}

// CloudProvider 云资产同步接口。
type CloudProvider interface {
	// SyncInstances 从云端同步主机实例列表。
	SyncInstances(accessKey, secretKey string, regions []string) ([]*model.Asset, error)
}
