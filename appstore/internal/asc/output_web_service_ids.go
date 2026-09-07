package asc

import "fmt"

// WebServiceIDMutationResult is the receipt for a verified private Developer
// Portal Services ID mutation.
type WebServiceIDMutationResult struct {
	Operation  string `json:"operation"`
	ServiceID  string `json:"serviceId"`
	Identifier string `json:"identifier,omitempty"`
	Name       string `json:"name,omitempty"`
	Changed    bool   `json:"changed"`
	Verified   bool   `json:"verified"`
	Status     string `json:"status"`
}

func webServiceIDMutationRows(result *WebServiceIDMutationResult) ([]string, [][]string) {
	headers := []string{"Operation", "Service ID", "Identifier", "Name", "Changed", "Verified", "Status"}
	if result == nil {
		return headers, nil
	}
	return headers, [][]string{{
		result.Operation,
		result.ServiceID,
		result.Identifier,
		result.Name,
		fmt.Sprintf("%t", result.Changed),
		fmt.Sprintf("%t", result.Verified),
		result.Status,
	}}
}
