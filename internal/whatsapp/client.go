package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	PhoneNumberID string
	WABAID        string
	AccessToken   string
	APIVersion    string
	HTTPClient    *http.Client
}

func NewClient(phoneNumberID, wabaID, accessToken, apiVersion string) *Client {
	return &Client{
		PhoneNumberID: phoneNumberID,
		WABAID:        wabaID,
		AccessToken:   accessToken,
		APIVersion:    apiVersion,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type SendMessageResponse struct {
	MessagingProduct string   `json:"messaging_product"`
	To               string   `json:"to"`
	Type             string   `json:"type"`
	Text             TextBody `json:"text"`
}

type TextBody struct {
	Body string `json:"body"`
}

type WhatsAppResponse struct {
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) SendMessage(ctx context.Context, to, messageType, content, mediaURL string) (string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", c.APIVersion, c.PhoneNumberID)

	var payload map[string]interface{}

	switch messageType {
	case "image":
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                NormalizePhoneNumber(to),
			"type":              "image",
			"image":             map[string]string{"link": mediaURL},
		}
	case "video":
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                NormalizePhoneNumber(to),
			"type":              "video",
			"video":             map[string]string{"link": mediaURL},
		}
	case "document":
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                NormalizePhoneNumber(to),
			"type":              "document",
			"document":          map[string]string{"link": mediaURL},
		}
	default:
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                NormalizePhoneNumber(to),
			"type":              "text",
			"text":              map[string]string{"body": content},
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result WhatsAppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != 200 && result.Error != nil {
		return "", fmt.Errorf("WhatsApp API error: %s", result.Error.Message)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	if len(result.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return result.Messages[0].ID, nil
}

func (c *Client) SendTemplateMessage(ctx context.Context, to, templateID string, params map[string]string) (string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", c.APIVersion, c.PhoneNumberID)

	components := make([]map[string]interface{}, 0)
	for key, value := range params {
		components = append(components, map[string]interface{}{
			"type": "body",
			"parameters": []map[string]string{
				{"type": "text", "parameter": key, "value": value},
			},
		})
	}

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                NormalizePhoneNumber(to),
		"type":              "template",
		"template": map[string]interface{}{
			"name":       templateID,
			"language":   map[string]string{"code": "en_US"},
			"components": components,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result WhatsAppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != 200 && result.Error != nil {
		return "", fmt.Errorf("WhatsApp API error: %s", result.Error.Message)
	}

	if len(result.Messages) > 0 {
		return result.Messages[0].ID, nil
	}

	return "", fmt.Errorf("failed to send template message: %s", string(respBody))
}
