package cloudsync

import (
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/logger"
)

type AliyunProvider struct{}

func NewAliyunProvider() *AliyunProvider {
	return &AliyunProvider{}
}

// getEndpointTemplate 获取阿里云 ECS API 地址模板，支持配置覆盖。
func getEndpointTemplate() string {
	cfg := config.Get()
	if cfg.Aliyun.ECSEndpoint != "" {
		return cfg.Aliyun.ECSEndpoint
	}
	return "ecs.%s.aliyuncs.com"
}

func (p *AliyunProvider) createClient(accessKey, secretKey, regionID string) (*ecs.Client, error) {
	cfg := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(secretKey),
		RegionId:        tea.String(regionID),
	}
	endpoint := fmt.Sprintf(getEndpointTemplate(), regionID)
	cfg.Endpoint = tea.String(endpoint)
	logger.Info("创建阿里云 ECS 客户端", zap.String("region", regionID), zap.String("endpoint", endpoint))
	return ecs.NewClient(cfg)
}

func (p *AliyunProvider) SyncInstances(accessKey, secretKey string, regions []string) ([]*model.Asset, error) {
	var allAssets []*model.Asset

	logger.Info("开始阿里云 ECS 同步", zap.Strings("regions", regions))

	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		client, err := p.createClient(accessKey, secretKey, region)
		if err != nil {
			logger.Error("创建阿里云客户端失败", zap.String("region", region), zap.Error(err))
			return nil, fmt.Errorf("创建阿里云客户端失败(%s): %w", region, err)
		}

		pageCount := 0
		nextToken := ""
		for {
			req := &ecs.DescribeInstancesRequest{
				RegionId:   tea.String(region),
				MaxResults: tea.Int32(100),
			}
			if nextToken != "" {
				req.NextToken = tea.String(nextToken)
			}

			resp, err := client.DescribeInstances(req)
			if err != nil {
				logger.Error("阿里云 DescribeInstances 调用失败",
					zap.String("region", region),
					zap.Error(err),
				)
				return nil, fmt.Errorf("查询实例失败(%s): %w", region, err)
			}
			pageCount++

			if resp.Body == nil || resp.Body.Instances == nil {
				logger.Info("阿里云返回空结果", zap.String("region", region), zap.Int("page", pageCount))
				break
			}

			instanceCount := len(resp.Body.Instances.Instance)
			totalCount := tea.Int32Value(resp.Body.TotalCount)
			logger.Info("阿里云 DescribeInstances 返回",
				zap.String("region", region),
				zap.Int("page", pageCount),
				zap.Int("instance_count", instanceCount),
				zap.Int32("total_count", totalCount),
				zap.String("request_id", tea.StringValue(resp.Body.RequestId)),
			)

			for _, inst := range resp.Body.Instances.Instance {
				asset := p.mapToAsset(inst, region)
				allAssets = append(allAssets, asset)
			}

			if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
				break
			}
			nextToken = *resp.Body.NextToken
		}

		logger.Info("Region 同步完成", zap.String("region", region), zap.Int("pages", pageCount))
	}

	logger.Info("阿里云 ECS 同步完成", zap.Int("total_assets", len(allAssets)))
	return allAssets, nil
}

func (p *AliyunProvider) mapToAsset(inst *ecs.DescribeInstancesResponseBodyInstancesInstance, region string) *model.Asset {
	asset := &model.Asset{
		CloudInstanceID: tea.StringValue(inst.InstanceId),
		Hostname:        tea.StringValue(inst.HostName),
		OS:              tea.StringValue(inst.OSName),
		OSVersion:       tea.StringValue(inst.OSType),
		CPUCores:        int(tea.Int32Value(inst.Cpu)),
		MemoryMB:        int(tea.Int32Value(inst.Memory)),
		Status:          mapAliyunStatus(tea.StringValue(inst.Status)),
		AssetType:       "server",
		Source:          "aliyun",
		IDC:             region,
		SN:              tea.StringValue(inst.SerialNumber),
	}

	// 公网 IP
	if inst.PublicIpAddress != nil && inst.PublicIpAddress.IpAddress != nil && len(inst.PublicIpAddress.IpAddress) > 0 {
		asset.IP = tea.StringValue(inst.PublicIpAddress.IpAddress[0])
	}
	// EIP
	if asset.IP == "" && inst.EipAddress != nil && inst.EipAddress.IpAddress != nil {
		asset.IP = tea.StringValue(inst.EipAddress.IpAddress)
	}

	// 内网 IP
	if inst.VpcAttributes != nil && inst.VpcAttributes.PrivateIpAddress != nil && inst.VpcAttributes.PrivateIpAddress.IpAddress != nil && len(inst.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		asset.InnerIP = tea.StringValue(inst.VpcAttributes.PrivateIpAddress.IpAddress[0])
	}

	// 如果没有公网 IP，用内网 IP 填充
	if asset.IP == "" {
		asset.IP = asset.InnerIP
	}
	if asset.Hostname == "" {
		asset.Hostname = asset.CloudInstanceID
	}

	return asset
}

func mapAliyunStatus(s string) string {
	switch s {
	case "Running":
		return "online"
	case "Stopped", "Stopping":
		return "offline"
	default:
		return "offline"
	}
}
