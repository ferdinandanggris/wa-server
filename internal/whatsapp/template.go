package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type MetaTemplate struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

type createTemplateResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Error    *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type listTemplatesResponse struct {
	Data   []MetaTemplate `json:"data"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
	Paging *struct {
		Cursors struct {
			Before string `json:"before"`
			After  string `json:"after"`
		} `json:"cursors"`
		Next string `json:"next"`
	} `json:"paging"`
}

type component struct {
	Type    string   `json:"type"`
	Format  string   `json:"format,omitempty"`
	Text    string   `json:"text,omitempty"`
	Buttons []button `json:"buttons,omitempty"`
}

type button struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (c *Client) CreateTemplate(ctx context.Context, name, language, category string, tmpl *TemplateComponents) (string, string, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates", 	c.APIVersion, c.WABAID)

	components := buildComponents(tmpl)

	payload := map[string]interface{}{
		"name":       name,
		"language":   language,
		"category":   category,
		"components": components,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result createTemplateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", "", fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != 200 && result.Error != nil {
		return "", "", fmt.Errorf("Meta API error: %s", result.Error.Message)
	}

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("Meta API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return result.ID, result.Status, nil
}

func (c *Client) GetTemplates(ctx context.Context) ([]MetaTemplate, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates", 	c.APIVersion, c.WABAID)

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

	var result listTemplatesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != 200 && result.Error != nil {
		return nil, fmt.Errorf("Meta API error: %s", result.Error.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Meta API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return result.Data, nil
}

func (c *Client) GetTemplateByName(ctx context.Context, name string) (*MetaTemplate, error) {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates?name=%s", 	c.APIVersion, c.WABAID, name)

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

	var result listTemplatesResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if resp.StatusCode != 200 && result.Error != nil {
		return nil, fmt.Errorf("Meta API error: %s", result.Error.Message)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Meta API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	if len(result.Data) == 0 {
		return nil, nil
	}

	return &result.Data[0], nil
}

func (c *Client) DeleteTemplate(ctx context.Context, name string) error {
	endpoint := fmt.Sprintf("https://graph.facebook.com/%s/%s/message_templates?name=%s", 	c.APIVersion, c.WABAID, name)

	req, err := http.NewRequestWithContext(ctx, "DELETE", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Meta API error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	return nil
}

type TemplateComponents struct {
	HeaderType    string
	HeaderContent string
	BodyContent   string
	FooterText    string
	ButtonsJSON   string
}

func buildComponents(tc *TemplateComponents) []component {
	var components []component

	if tc.HeaderType != "" {
		header := component{
			Type: "HEADER",
		}
		switch tc.HeaderType {
		case "TEXT":
			header.Format = "TEXT"
			header.Text = tc.HeaderContent
		case "IMAGE":
			header.Format = "IMAGE"
		case "VIDEO":
			header.Format = "VIDEO"
		case "DOCUMENT":
			header.Format = "DOCUMENT"
		}
		components = append(components, header)
	}

	body := component{
		Type: "BODY",
		Text: tc.BodyContent,
	}
	components = append(components, body)

	if tc.FooterText != "" {
		footer := component{
			Type: "FOOTER",
			Text: tc.FooterText,
		}
		components = append(components, footer)
	}

	if tc.ButtonsJSON != "" {
		var buttons []button
		if err := json.Unmarshal([]byte(tc.ButtonsJSON), &buttons); err == nil && len(buttons) > 0 {
			btns := component{
				Type:    "BUTTONS",
				Buttons: buttons,
			}
			components = append(components, btns)
		}
	}

	return components
}
