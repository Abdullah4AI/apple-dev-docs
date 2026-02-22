package asc

// DevicePlatform represents the platform of a device.
type DevicePlatform string

const (
	DevicePlatformIOS   DevicePlatform = "IOS"
	DevicePlatformMacOS DevicePlatform = "MAC_OS"
)

// DeviceStatus represents the status of a device.
type DeviceStatus string

const (
	DeviceStatusEnabled  DeviceStatus = "ENABLED"
	DeviceStatusDisabled DeviceStatus = "DISABLED"
)

// DeviceClass represents the device class reported by ASC.
type DeviceClass string

const (
	DeviceClassAppleWatch DeviceClass = "APPLE_WATCH"
	DeviceClassIPad       DeviceClass = "IPAD"
	DeviceClassIPhone     DeviceClass = "IPHONE"
	DeviceClassIPod       DeviceClass = "IPOD"
	DeviceClassAppleTV    DeviceClass = "APPLE_TV"
	DeviceClassMac        DeviceClass = "MAC"
)

// DeviceAttributes describes an App Store Connect device.
type DeviceAttributes struct {
	Name        string         `json:"name"`
	Platform    DevicePlatform `json:"platform"`
	UDID        string         `json:"udid"`
	DeviceClass DeviceClass    `json:"deviceClass,omitempty"`
	Status      DeviceStatus   `json:"status,omitempty"`
	Model       string         `json:"model,omitempty"`
	AddedDate   string         `json:"addedDate,omitempty"`
}

// DevicesResponse is the response from devices list endpoint.
type DevicesResponse = Response[DeviceAttributes]

// DeviceResponse is the response from device detail endpoint.
type DeviceResponse = SingleResponse[DeviceAttributes]

// DeviceCreateAttributes describes attributes for creating a device.
type DeviceCreateAttributes struct {
	Name     string         `json:"name"`
	UDID     string         `json:"udid"`
	Platform DevicePlatform `json:"platform"`
}

// DeviceCreateData is the data portion of a device create request.
type DeviceCreateData struct {
	Type       ResourceType           `json:"type"`
	Attributes DeviceCreateAttributes `json:"attributes"`
}

// DeviceCreateRequest is a request to create a device.
type DeviceCreateRequest struct {
	Data DeviceCreateData `json:"data"`
}

// DeviceUpdateAttributes describes attributes for updating a device.
type DeviceUpdateAttributes struct {
	Name   *string       `json:"name,omitempty"`
	Status *DeviceStatus `json:"status,omitempty"`
}

// DeviceUpdateData is the data portion of a device update request.
type DeviceUpdateData struct {
	Type       ResourceType            `json:"type"`
	ID         string                  `json:"id"`
	Attributes *DeviceUpdateAttributes `json:"attributes,omitempty"`
}

// DeviceUpdateRequest is a request to update a device.
type DeviceUpdateRequest struct {
	Data DeviceUpdateData `json:"data"`
}
