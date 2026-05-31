package app

type AppOverview struct {
	ProductName    string         `json:"productName"`
	Tagline        string         `json:"tagline"`
	CurrentPhase   string         `json:"currentPhase"`
	FrontendStack  string         `json:"frontendStack"`
	BackendStack   string         `json:"backendStack"`
	DataStack      string         `json:"dataStack"`
	Highlights     []string       `json:"highlights"`
	Milestones     []OverviewItem `json:"milestones"`
	Architecture   []OverviewItem `json:"architecture"`
	NextActions    []string       `json:"nextActions"`
	AcceptanceBars []string       `json:"acceptanceBars"`
}

type OverviewItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type OverviewService struct{}

func NewOverviewService() *OverviewService {
	return &OverviewService{}
}

func (s *OverviewService) GetOverview() AppOverview {
	return AppOverview{
		ProductName:   "StreamSignal",
		Tagline:       "Send the signal. Go live everywhere.",
		CurrentPhase:  "Milestone 8 is complete. Milestone 9 is now focused on a full security audit and high-value remediation work.",
		FrontendStack: "React + TypeScript via Wails",
		BackendStack:  "Go application services and Wails bindings",
		DataStack:     "SQLite for app data, Windows Credential Manager for secrets",
		Highlights: []string{
			"All business logic stays in Go.",
			"The frontend is a thin shell for forms, previews, confirmations, and results.",
			"Preview and Dry Run must stay completely network-free.",
			"Test Mode must always redirect away from production destinations.",
		},
		Milestones: []OverviewItem{
			{
				Title:       "Milestone 0",
				Description: "Bootstrap Wails, React, and the first backend/frontend handshake.",
				Status:      "done",
			},
			{
				Title:       "Milestone 1",
				Description: "Add domain models, repository interfaces, SQLite wiring, persistence, and CRUD-ready backend services.",
				Status:      "done",
			},
			{
				Title:       "Milestone 2",
				Description: "Implement template rendering, validation, and preview generation.",
				Status:      "done",
			},
			{
				Title:       "Milestone 3",
				Description: "Build configuration screens for destinations, settings, and logs.",
				Status:      "done",
			},
			{
				Title:       "Milestone 4",
				Description: "Implement execution orchestration for Dry Run, Go Live, and result handling.",
				Status:      "done",
			},
			{
				Title:       "Milestone 5",
				Description: "Connect real posting services for Discord, Bluesky, and Mastodon.",
				Status:      "done",
			},
			{
				Title:       "Milestone 6",
				Description: "Implement the Bluesky Live Now status lifecycle: set on Go Live, clear on End Stream, and support safe manual recovery.",
				Status:      "done",
			},
			{
				Title:       "Milestone 7",
				Description: "Harden the MVP with clearer UX, broader coverage, and final workflow polish.",
				Status:      "done",
			},
			{
				Title:       "Milestone 8",
				Description: "Validate the test strategy against best practice and close any high-value coverage or test-quality gaps.",
				Status:      "done",
			},
			{
				Title:       "Milestone 9",
				Description: "Run a full security audit, prioritize findings, and implement the most important remediations.",
				Status:      "in_progress",
			},
		},
		Architecture: []OverviewItem{
			{
				Title:       "Frontend",
				Description: "Feature screens call Wails bindings and render typed backend responses.",
				Status:      "ready",
			},
			{
				Title:       "Application Layer",
				Description: "Workflow services will orchestrate Preview, Dry Run, Go Live, and End Stream.",
				Status:      "ready_for_integrations",
			},
			{
				Title:       "Domain + Ports",
				Description: "Core models and interfaces define the contract for storage and publishers.",
				Status:      "ready",
			},
		},
		NextActions: []string{
			"Verify the GitHub-hosted quality gate in live repository runs once the public repo is active.",
			"Continue the security pass on diagnostics depth, recovery behavior, and future adapter additions.",
			"Use the test-validation baseline to verify security changes without weakening workflow safety.",
		},
		AcceptanceBars: []string{
			"Core workflows stay testable without launching the UI.",
			"Secrets never land in SQLite or plaintext config files.",
			"Platform integrations remain behind interfaces and are mockable.",
		},
	}
}
