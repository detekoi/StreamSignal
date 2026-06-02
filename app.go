package main

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	appsvc "StreamSignal/internal/app"
	"StreamSignal/internal/domain"
	"StreamSignal/internal/platforms/bluesky"
	"StreamSignal/internal/platforms/discord"
	"StreamSignal/internal/platforms/mastodon"
	"StreamSignal/internal/ports"
	"StreamSignal/internal/secrets/wincred"
	"StreamSignal/internal/storage/secure"
	"StreamSignal/internal/storage/sqlite"
)

// App exposes Wails bindings for the frontend.
type App struct {
	ctx             context.Context
	db              *sql.DB
	settingsService *appsvc.SettingsService
	destinationSvc  *appsvc.DestinationService
	credentialSvc   *appsvc.CredentialSetupService
	previewService  *appsvc.PreviewService
	logService      *appsvc.LogService
	diagnosticsSvc  *appsvc.DiagnosticsService
	executionSvc    *appsvc.ExecutionService
	recoverySvc     *appsvc.LiveNowRecoveryService
}

// NewApp creates a new application shell.
func NewApp() *App {
	dataDir, err := appsvc.EnsureDataDir()
	if err != nil {
		panic(err)
	}

	return newAppWithDependencies(filepath.Join(dataDir, "streamsignal.db"), wincred.NewStore())
}

func newAppWithDatabasePath(databasePath string) *App {
	return newAppWithDependencies(databasePath, wincred.NewStore())
}

func newAppWithDependencies(databasePath string, secretStore ports.SecretStore) *App {
	db, err := sqlite.Open(databasePath)
	if err != nil {
		panic(err)
	}

	settingsRepo := secure.NewSettingsRepository(sqlite.NewSettingsRepository(db), secretStore)
	destinationRepo := secure.NewDestinationRepository(sqlite.NewDestinationRepository(db), secretStore)
	logRepo := sqlite.NewLogRepository(db)
	historyRepo := sqlite.NewPostHistoryRepository(db)
	liveNowSessionRepo := secure.NewLiveNowSessionRepository(sqlite.NewLiveNowSessionRepository(db), secretStore)
	discordPublisher := discord.NewPublisher(nil)
	blueskyPublisher := bluesky.NewPublisher("", nil)
	mastodonPublisher := mastodon.NewPublisher(nil)

	return &App{
		db:              db,
		settingsService: appsvc.NewSettingsService(settingsRepo),
		destinationSvc:  appsvc.NewDestinationService(destinationRepo),
		credentialSvc:   appsvc.NewCredentialSetupService(discordPublisher, blueskyPublisher, mastodonPublisher),
		previewService:  appsvc.NewPreviewService(destinationRepo, settingsRepo, liveNowSessionRepo),
		logService:      appsvc.NewLogService(logRepo),
		diagnosticsSvc:  appsvc.NewDiagnosticsService(settingsRepo, destinationRepo, logRepo, liveNowSessionRepo),
		executionSvc: appsvc.NewExecutionService(
			destinationRepo,
			settingsRepo,
			historyRepo,
			liveNowSessionRepo,
			discordPublisher,
			blueskyPublisher,
			mastodonPublisher,
		),
		recoverySvc: appsvc.NewLiveNowRecoveryService(destinationRepo, liveNowSessionRepo, blueskyPublisher),
	}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown closes open resources when the app exits.
func (a *App) shutdown(context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

func (a *App) GetSettings() (domain.AppSettings, error) {
	return a.settingsService.Load(a.ctx)
}

func (a *App) SaveSettings(settings domain.AppSettings) (domain.AppSettings, error) {
	saved, err := a.settingsService.Save(a.ctx, settings)
	if err == nil {
		_ = a.logService.Append(a.ctx, "settings", "save_settings", "SUCCESS", "Saved application settings.")
	}
	return saved, err
}

func (a *App) ListDestinations() ([]domain.Destination, error) {
	return a.destinationSvc.List(a.ctx)
}

func (a *App) SaveDestination(destination domain.Destination) (domain.Destination, error) {
	saved, err := a.destinationSvc.Save(a.ctx, destination)
	if err == nil {
		_ = a.logService.Append(a.ctx, saved.Name, "save_destination", "SUCCESS", "Saved destination configuration.")
	}
	return saved, err
}

func (a *App) DeleteDestination(id string) error {
	err := a.destinationSvc.Delete(a.ctx, id)
	if err == nil {
		_ = a.logService.Append(a.ctx, id, "delete_destination", "SUCCESS", "Deleted destination configuration.")
	}
	return err
}

func (a *App) TestDestinationConnection(destination domain.Destination) domain.CredentialCheckResult {
	result := a.credentialSvc.TestDestination(a.ctx, destination)
	status := result.State
	if status == "" {
		status = "FAILED"
	}
	_ = a.logService.Append(a.ctx, destination.Name, "test_destination_connection", status, result.Message)
	return result
}

func (a *App) GeneratePreview(announcement domain.Announcement) ([]domain.PreviewItem, error) {
	items, err := a.previewService.Generate(a.ctx, announcement)
	if err == nil {
		_ = a.logService.Append(a.ctx, "preview", "generate_preview", "SUCCESS", fmt.Sprintf("Generated %d preview items.", len(items)))
	}
	return items, err
}

func (a *App) GetLogs() ([]domain.LogEntry, error) {
	return a.logService.ListRecent(a.ctx, 100)
}

func (a *App) GetDiagnostics() (string, error) {
	return a.diagnosticsSvc.Build(a.ctx)
}

func (a *App) GoLive(announcement domain.Announcement) (domain.ExecutionSummary, error) {
	summary, err := a.executionSvc.GoLive(a.ctx, announcement)
	if err == nil {
		_ = a.logService.Append(a.ctx, "go_live", "go_live", executionLogStatus(summary), executionLogMessage("Go Live", summary))
	}
	return summary, err
}

func (a *App) ForceGoLive(announcement domain.Announcement) (domain.ExecutionSummary, error) {
	summary, err := a.executionSvc.ForceGoLive(a.ctx, announcement)
	if err == nil {
		_ = a.logService.Append(a.ctx, "go_live", "force_go_live", executionLogStatus(summary), executionLogMessage("Confirmed Go Live", summary))
	}
	return summary, err
}

func (a *App) EndStream(announcement domain.Announcement) (domain.ExecutionSummary, error) {
	summary, err := a.executionSvc.EndStream(a.ctx, announcement)
	if err == nil {
		_ = a.logService.Append(a.ctx, "end_stream", "end_stream", executionLogStatus(summary), executionLogMessage("End Stream", summary))
	}
	return summary, err
}

func (a *App) ListPendingLiveNowSessions() ([]domain.ActiveLiveNowSession, error) {
	return a.recoverySvc.ListPending(a.ctx)
}

func (a *App) ClearPendingLiveNowSession(destinationID string) (domain.ExecutionResult, error) {
	result, err := a.recoverySvc.ClearPending(a.ctx, destinationID)
	if err != nil {
		_ = a.logService.Append(
			a.ctx,
			destinationID,
			"clear_pending_live_now",
			"FAILED",
			fmt.Sprintf("Live Now recovery failed: %s", err.Error()),
		)
		return result, err
	}
	destinationName := result.DestinationName
	if destinationName == "" {
		destinationName = destinationID
	}
	_ = a.logService.Append(a.ctx, destinationName, "clear_pending_live_now", string(result.State), result.Message)
	return result, err
}

func executionLogStatus(summary domain.ExecutionSummary) string {
	if summary.Status == "" {
		return string(domain.ExecutionSummaryStatusSuccess)
	}
	return string(summary.Status)
}

func executionLogMessage(action string, summary domain.ExecutionSummary) string {
	message := fmt.Sprintf(
		"%s returned %d results: %d success, %d failed, %d skipped, %d validation.",
		action,
		summary.TotalCount,
		summary.SuccessCount,
		summary.FailedCount,
		summary.SkippedCount,
		summary.ValidationErrorCount,
	)
	if summary.RequiresDuplicateConfirmation {
		return fmt.Sprintf("%s Duplicate confirmation required.", message)
	}
	return message
}
