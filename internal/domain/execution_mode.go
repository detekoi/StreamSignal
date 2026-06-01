package domain

type ExecutionMode string

const (
	ExecutionModeGoLive    ExecutionMode = "go_live"
	ExecutionModeEndStream ExecutionMode = "end_stream"
)
