package asc

// BetaCrashLogAttributes describes a beta crash log resource.
type BetaCrashLogAttributes struct {
	LogText string `json:"logText,omitempty"`
}

// BetaCrashLogResponse is the response from beta crash log endpoints.
type BetaCrashLogResponse = SingleResponse[BetaCrashLogAttributes]

// BetaFeedbackCrashSubmissionResponse is the response from crash submission detail endpoint.
type BetaFeedbackCrashSubmissionResponse = SingleResponse[CrashAttributes]

// BetaFeedbackScreenshotSubmissionResponse is the response from screenshot submission detail endpoint.
type BetaFeedbackScreenshotSubmissionResponse = SingleResponse[FeedbackAttributes]
