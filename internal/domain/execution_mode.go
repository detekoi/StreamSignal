package domain

type ExecutionMode string

const (
	ExecutionModeDryRun    ExecutionMode = "dry_run"
	ExecutionModeGoLive    ExecutionMode = "go_live"
	ExecutionModeEndStream ExecutionMode = "end_stream"
)
