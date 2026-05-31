package domain

type ExecutionState string
type ExecutionSummaryStatus string

const (
	ExecutionStateSuccess         ExecutionState = "SUCCESS"
	ExecutionStateFailed          ExecutionState = "FAILED"
	ExecutionStateSkipped         ExecutionState = "SKIPPED"
	ExecutionStateValidationError ExecutionState = "VALIDATION_ERROR"

	ExecutionSummaryStatusSuccess ExecutionSummaryStatus = "SUCCESS"
	ExecutionSummaryStatusWarning ExecutionSummaryStatus = "WARNING"
	ExecutionSummaryStatusPartial ExecutionSummaryStatus = "PARTIAL"
)

type ExecutionResult struct {
	DestinationID   string              `json:"destinationID"`
	DestinationName string              `json:"destinationName"`
	Platform        DestinationPlatform `json:"platform"`
	State           ExecutionState      `json:"state"`
	Message         string              `json:"message"`
	Content         string              `json:"content"`
}

type ExecutionSummary struct {
	Mode                          ExecutionMode          `json:"mode"`
	Status                        ExecutionSummaryStatus `json:"status"`
	TestModeActive                bool                   `json:"testModeActive"`
	Results                       []ExecutionResult      `json:"results"`
	TotalCount                    int                    `json:"totalCount"`
	SuccessCount                  int                    `json:"successCount"`
	FailedCount                   int                    `json:"failedCount"`
	SkippedCount                  int                    `json:"skippedCount"`
	ValidationErrorCount          int                    `json:"validationErrorCount"`
	RequiresDuplicateConfirmation bool                   `json:"requiresDuplicateConfirmation"`
	DuplicateWarningMessage       string                 `json:"duplicateWarningMessage"`
}
