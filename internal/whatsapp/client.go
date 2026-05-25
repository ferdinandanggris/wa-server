package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/wa-server/internal/models"
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

func (c *Client) GetPricingAnalytics(ctx context.Context, start, end int64) (*models.PricingAnalyticsResponse, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s", c.APIVersion, c.WABAID)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	fields := fmt.Sprintf("pricing_analytics.start(%d).end(%d).granularity(DAILY).phone_numbers([]).dimensions([\"PHONE\",\"PRICING_CATEGORY\"])", start, end)
	q.Set("fields", fields)
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		PricingAnalytics *models.PricingAnalyticsResponse `json:"pricing_analytics"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.PricingAnalytics == nil {
		return nil, fmt.Errorf("no pricing_analytics in response")
	}

	return result.PricingAnalytics, nil
}

// GetConversationAnalytics calls Meta's conversation_analytics endpoint for cost data.
func (c *Client) GetConversationAnalytics(ctx context.Context, start, end time.Time, granularity string) (*models.ConversationAnalyticsResponse, error) {
	if granularity == "" {
		granularity = "DAY"
	}

	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s", c.APIVersion, c.WABAID)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("fields", "conversation_analytics")
	q.Set("start", start.Format(time.RFC3339))
	q.Set("end", end.Format(time.RFC3339))
	q.Set("granularity", granularity)
	q.Set("phone_numbers", "[]")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		ConversationAnalytics *models.ConversationAnalyticsResponse `json:"conversation_analytics"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.ConversationAnalytics == nil {
		return nil, fmt.Errorf("no conversation_analytics in response")
	}

	return result.ConversationAnalytics, nil
}

type phoneNumbersResponse struct {
	Data []models.WhatsAppPhoneNumber `json:"data"`
}

func (c *Client) GetPhoneNumbers(ctx context.Context) ([]models.WhatsAppPhoneNumber, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/phone_numbers", c.APIVersion, c.WABAID)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result phoneNumbersResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result.Data, nil
}

func (c *Client) GetBusinessProfile(ctx context.Context, phoneNumberID string) (*models.WhatsAppBusinessProfile, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/whatsapp_business_profile", c.APIVersion, phoneNumberID)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("fields", "about,address,description,email,profile_picture_url,websites,vertical")
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result models.WhatsAppBusinessProfileResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no business profile data found")
	}

	return &result.Data[0], nil
}

func (c *Client) UpdateBusinessProfile(ctx context.Context, phoneNumberID string, profile *models.WhatsAppBusinessProfile) error {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/whatsapp_business_profile", c.APIVersion, phoneNumberID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
	}
	if profile.About != "" {
		payload["about"] = profile.About
	}
	if profile.Address != "" {
		payload["address"] = profile.Address
	}
	if profile.Description != "" {
		payload["description"] = profile.Description
	}
	if profile.Email != "" {
		payload["email"] = profile.Email
	}
	if profile.Vertical != "" {
		payload["vertical"] = profile.Vertical
	}
	if len(profile.Websites) > 0 {
		websitesJSON, _ := json.Marshal(profile.Websites)
		payload["websites"] = string(websitesJSON)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("WhatsApp API returned success=false")
	}

	return nil
}

func (c *Client) SendMessageFromPhone(ctx context.Context, phoneNumberID, to, messageType, content, mediaURL, contextMsgID string) (string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", c.APIVersion, phoneNumberID)

	var payload map[string]interface{}

	switch messageType {
	case "image":
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                to,
			"type":              "image",
			"image": map[string]string{
				"link": mediaURL,
			},
		}
	case "audio":
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                to,
			"type":              "audio",
			"audio": map[string]string{
				"id": mediaURL,
			},
		}
	case "video":
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                to,
			"type":              "video",
			"video": map[string]string{
				"link": mediaURL,
			},
		}
	default:
		payload = map[string]interface{}{
			"messaging_product": "whatsapp",
			"to":                to,
			"type":              "text",
			"text": map[string]string{
				"body": content,
			},
		}
	}

	if contextMsgID != "" {
		payload["context"] = map[string]string{
			"message_id": contextMsgID,
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return "", fmt.Errorf("WhatsApp API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var waResp WhatsAppResponse
	if err := json.Unmarshal(respBody, &waResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if waResp.Error != nil {
		return "", fmt.Errorf("WhatsApp API error: %s", waResp.Error.Message)
	}

	if len(waResp.Messages) == 0 {
		return "", fmt.Errorf("no message ID in response")
	}

	return waResp.Messages[0].ID, nil
}

func (c *Client) SendMessage(ctx context.Context, to, messageType, content, mediaURL, contextMsgID string) (string, error) {
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

	if contextMsgID != "" {
		payload["context"] = map[string]string{
			"message_id": contextMsgID,
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

func (c *Client) SendTemplateMessageFromPhone(ctx context.Context, phoneNumberID, to, templateID string, params map[string]string) (string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", c.APIVersion, phoneNumberID)

	result, err := c.sendTemplate(ctx, endpoint, to, templateID, params)
	if err != nil {
		return "", err
	}
	return result.Messages[0].ID, nil
}

func (c *Client) SendTemplateMessage(ctx context.Context, to, templateID string, params map[string]string) (string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/messages", c.APIVersion, c.PhoneNumberID)

	result, err := c.sendTemplate(ctx, endpoint, to, templateID, params)
	if err != nil {
		return "", err
	}
	return result.Messages[0].ID, nil
}

func (c *Client) sendTemplate(ctx context.Context, endpoint, to, templateID string, params map[string]string) (*WhatsAppResponse, error) {
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
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result WhatsAppResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != 200 && result.Error != nil {
		return nil, fmt.Errorf("WhatsApp API error: %s", result.Error.Message)
	}

	if len(result.Messages) > 0 {
		return &result, nil
	}

	return nil, fmt.Errorf("failed to send template message: %s", string(respBody))
}
