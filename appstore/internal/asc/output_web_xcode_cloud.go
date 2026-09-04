package asc

// WebXcodeCloudWorkflowsListResult is the computed list of Xcode Cloud workflows
// for a product. The web CI list model only exposes id, name, and description.
type WebXcodeCloudWorkflowsListResult struct {
	ProductID string                          `json:"productId"`
	Workflows []WebXcodeCloudWorkflowListItem `json:"workflows"`
}

// WebXcodeCloudWorkflowListItem is one workflow from the web CI list endpoint.
type WebXcodeCloudWorkflowListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func webXcodeCloudWorkflowsListRows(result *WebXcodeCloudWorkflowsListResult) ([]string, [][]string) {
	headers := []string{"Workflow ID", "Name", "Description"}
	if result == nil {
		return headers, nil
	}
	rows := make([][]string, 0, len(result.Workflows))
	for _, item := range result.Workflows {
		rows = append(rows, []string{item.ID, item.Name, item.Description})
	}
	return headers, rows
}
