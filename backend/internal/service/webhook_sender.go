package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var webhookHTTPClient = &http.Client{Timeout: 10 * time.Second}

// WebhookTarget 单个渠道的 Webhook 配置，内嵌在业务对象（告警规则/流水线/工单模板）中。
type WebhookTarget struct {
	WebhookURL string `json:"webhook_url"`
	Secret     string `json:"secret"`
}

// ParseNotifyConfig 从 JSON 字符串解析 notify_config。
func ParseNotifyConfig(raw string) map[string]WebhookTarget {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var cfg map[string]WebhookTarget
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return nil
	}
	// 过滤空 URL
	result := make(map[string]WebhookTarget)
	for k, v := range cfg {
		if strings.TrimSpace(v.WebhookURL) != "" {
			result[k] = v
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// SendDingtalk 发送钉钉机器人 Markdown 消息。
// 文档: https://open.dingtalk.com/document/orgapp/custom-bot-access-send-message
func SendDingtalk(webhookURL, secret, title, markdown string) error {
	finalURL := webhookURL
	if secret != "" {
		ts := fmt.Sprintf("%d", time.Now().UnixMilli())
		sign := dingtalkSign(ts, secret)
		sep := "&"
		if !strings.Contains(webhookURL, "?") {
			sep = "?"
		}
		finalURL = fmt.Sprintf("%s%stimestamp=%s&sign=%s", webhookURL, sep, ts, url.QueryEscape(sign))
	}

	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  markdown,
		},
	}
	return postJSON(finalURL, body)
}

// SendLark 发送飞书机器人 Markdown 卡片消息。
// 文档: https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
func SendLark(webhookURL, secret, title, markdown string) error {
	card := map[string]interface{}{
		"msg_type": "interactive",
		"card": map[string]interface{}{
			"header": map[string]interface{}{
				"title": map[string]string{
					"tag":     "plain_text",
					"content": title,
				},
			},
			"elements": []map[string]interface{}{
				{
					"tag":     "markdown",
					"content": markdown,
				},
			},
		},
	}

	if secret != "" {
		ts := fmt.Sprintf("%d", time.Now().Unix())
		sign := larkSign(ts, secret)
		card["timestamp"] = ts
		card["sign"] = sign
	}

	return postJSON(webhookURL, card)
}

// SendWecom 发送企业微信机器人 Markdown 消息。
// 文档: https://developer.work.weixin.qq.com/document/path/91770
func SendWecom(webhookURL, _, _, markdown string) error {
	body := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": markdown,
		},
	}
	return postJSON(webhookURL, body)
}

// SendGenericWebhook 发送通用 Webhook（JSON POST + 可选 HMAC 签名）。
func SendGenericWebhook(webhookURL, secret, title, markdown string) error {
	body := map[string]interface{}{
		"title":     title,
		"content":   markdown,
		"timestamp": time.Now().Unix(),
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", webhookURL, strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(data)
		req.Header.Set("X-Signature", fmt.Sprintf("sha256=%s", base64.StdEncoding.EncodeToString(mac.Sum(nil))))
	}

	resp, err := webhookHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("webhook responded %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// DispatchWebhook 根据渠道类型路由到具体发送函数。
func DispatchWebhook(channelType, webhookURL, secret, title, markdown string) error {
	switch channelType {
	case "dingtalk":
		return SendDingtalk(webhookURL, secret, title, markdown)
	case "lark":
		return SendLark(webhookURL, secret, title, markdown)
	case "wecom":
		return SendWecom(webhookURL, secret, title, markdown)
	default:
		return SendGenericWebhook(webhookURL, secret, title, markdown)
	}
}

// --- 签名辅助 ---

func dingtalkSign(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func larkSign(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	mac.Write([]byte{})
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func postJSON(url string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := webhookHTTPClient.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s responded %d: %s", url, resp.StatusCode, string(respBody))
	}
	return nil
}
