package cloudsync

import (
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"

	"github.com/bigops/platform/internal/model"
)

type AliyunProvider struct{}

func NewAliyunProvider() *AliyunProvider {
	return &AliyunProvider{}
}

func (p *AliyunProvider) createClient(accessKey, secretKey, regionID string) (*ecs.Client, error) {
	cfg := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(secretKey),
		RegionId:        tea.String(regionID),
	}
	cfg.Endpoint = tea.String(fmt.Sprintf("ecs.%s.aliyuncs.com", regionID))
	return ecs.NewClient(cfg)
}

func (p *AliyunProvider) SyncInstances(accessKey, secretKey string, regions []string) ([]*model.Asset, error) {
	var allAssets []*model.Asset

	for _, region := range regions {
		region = strings.TrimSpace(region)
		if region == "" {
			continue
		}
		client, err := p.createClient(accessKey, secretKey, region)
		if err != nil {
			return nil, fmt.Errorf("创建阿里云客户端失败(%s): %w", region, err)
		}

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
				return nil, fmt.Errorf("查询实例失败(%s): %w", region, err)
			}

			if resp.Body == nil || resp.Body.Instances == nil {
				break
			}

			for _, inst := range resp.Body.Instances.Instance {
				asset := p.mapToAsset(inst, region)
				allAssets = append(allAssets, asset)
			}

			if resp.Body.NextToken == nil || *resp.Body.NextToken == "" {
				break
			}
			nextToken = *resp.Body.NextToken
		}
	}

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
