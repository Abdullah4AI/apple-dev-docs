package validation

func hasCheckID(checks []CheckResult, id string) bool {
	for _, check := range checks {
		if check.ID == id {
			return true
		}
	}
	return false
}
