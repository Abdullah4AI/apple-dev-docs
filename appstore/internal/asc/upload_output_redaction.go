package asc

// RedactUploadOperations returns a presentation-safe copy of presigned upload
// operations. URLs and request-header values are capabilities; method, size,
// offset, expiration, and header names remain useful for dry-run diagnostics.
func RedactUploadOperations(operations []UploadOperation) []UploadOperation {
	if operations == nil {
		return nil
	}

	safe := make([]UploadOperation, len(operations))
	for index, operation := range operations {
		safe[index] = operation
		safe[index].URL = RedactedValuePlaceholder
		if operation.RequestHeaders != nil {
			safe[index].RequestHeaders = make([]HTTPHeader, len(operation.RequestHeaders))
			for headerIndex, header := range operation.RequestHeaders {
				safe[index].RequestHeaders[headerIndex] = header
				safe[index].RequestHeaders[headerIndex].Value = RedactedValuePlaceholder
			}
		}
	}
	return safe
}
