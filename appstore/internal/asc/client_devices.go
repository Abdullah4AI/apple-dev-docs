package asc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// GetDevices retrieves the list of devices.
func (c *Client) GetDevices(ctx context.Context, opts ...DevicesOption) (*DevicesResponse, error) {
	query := &devicesQuery{}
	for _, opt := range opts {
		opt(query)
	}

	path := "/v1/devices"
	if query.nextURL != "" {
		// Validate nextURL to prevent credential exfiltration
		if err := validateNextURL(query.nextURL); err != nil {
			return nil, fmt.Errorf("devices: %w", err)
		}
		path = query.nextURL
	} else if queryString := buildDevicesQuery(query); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response DevicesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetDevice retrieves a single device by ID.
func (c *Client) GetDevice(ctx context.Context, deviceID string, fields []string) (*DeviceResponse, error) {
	deviceID = strings.TrimSpace(deviceID)
	path := fmt.Sprintf("/v1/devices/%s", deviceID)
	if queryString := buildDevicesFieldsQuery(fields); queryString != "" {
		path += "?" + queryString
	}

	data, err := c.do(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}

	var response DeviceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// CreateDevice registers a new device.
func (c *Client) CreateDevice(ctx context.Context, attrs DeviceCreateAttributes) (*DeviceResponse, error) {
	payload := DeviceCreateRequest{
		Data: DeviceCreateData{
			Type:       ResourceTypeDevices,
			Attributes: attrs,
		},
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, "POST", "/v1/devices", body)
	if err != nil {
		return nil, err
	}

	var response DeviceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// UpdateDevice updates a device by ID.
func (c *Client) UpdateDevice(ctx context.Context, deviceID string, attrs DeviceUpdateAttributes) (*DeviceResponse, error) {
	deviceID = strings.TrimSpace(deviceID)
	payload := DeviceUpdateRequest{
		Data: DeviceUpdateData{
			Type: ResourceTypeDevices,
			ID:   deviceID,
		},
	}
	if attrs.Name != nil || attrs.Status != nil {
		payload.Data.Attributes = &attrs
	}

	body, err := BuildRequestBody(payload)
	if err != nil {
		return nil, err
	}

	data, err := c.do(ctx, "PATCH", fmt.Sprintf("/v1/devices/%s", deviceID), body)
	if err != nil {
		return nil, err
	}

	var response DeviceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
