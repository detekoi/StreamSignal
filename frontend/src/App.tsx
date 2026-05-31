import { FormEvent, useEffect, useState } from 'react';
import './App.css';
import {
    clearPendingLiveNowSession,
    deleteDestination,
    dryRun,
    endStream,
    forceGoLive,
    generatePreview,
    getDiagnostics,
    getLogs,
    goLive,
    getOverview,
    listPendingLiveNowSessions,
    getSettings,
    listDestinations,
    saveDestination,
    saveSettings,
} from './lib/api/streamsignal';
import type { AppOverview } from './types/app-overview';
import {
    createEmptyDestinationForm,
    toDestinationFormState,
    toDestinationInput,
    type DestinationFormState,
    type DestinationInput,
} from './types/destination';
import type { ExecutionSummary } from './types/execution';
import type { ActiveLiveNowSession } from './types/live-now-recovery';
import type { AnnouncementInput, PreviewItem } from './types/preview';
import type { LogEntry } from './types/log-entry';
import type { AppSettings } from './types/settings';

type TabKey = 'home' | 'destinations' | 'settings' | 'logs';

const STORED_SECRET_TOKEN = '[stored securely]';

const initialAnnouncement: AnnouncementInput = {
    streamTitle: '',
    streamURL: '',
    category: '',
    message: '',
    hashtags: '',
};

const initialDestination = createEmptyDestinationForm();

const defaultSettings: AppSettings = {
    testModeEnabled: false,
    testDiscordWebhookKey: '',
    testBlueskyAccountIdentifier: '',
    testBlueskyCredentialKey: '',
    testMastodonCredentialKey: '',
    testMastodonInstanceURL: '',
    defaultStreamURL: '',
    defaultHashtags: '',
    duplicateProtectionEnabled: true,
    duplicateWindowMinutes: 10,
    endStreamPostEnabled: false,
    endStreamTemplate: '',
};

type SecretSettingsCache = Pick<AppSettings, 'testDiscordWebhookKey' | 'testBlueskyCredentialKey' | 'testMastodonCredentialKey'>;

type DestinationSecretCache = Pick<DestinationFormState, 'discordWebhookKey' | 'blueskyCredentialKey' | 'mastodonCredentialKey'>;

function executionModeLabel(mode: ExecutionSummary['mode']) {
    switch (mode) {
        case 'dry_run':
            return 'Dry Run';
        case 'go_live':
            return 'Go Live';
        case 'end_stream':
            return 'End Stream';
        default:
            return 'Execution';
    }
}

function executionResultKind(mode: ExecutionSummary['mode'], message: string) {
    if (mode === 'end_stream') {
        if (message.includes('Live Now cleared') || message.includes('Recovered and cleared pending Live Now session')) {
            return 'Live Now Clear';
        }
        if (message.includes('End stream post')) {
            return 'End Stream Post';
        }
        return 'End Stream Action';
    }
    if (mode === 'go_live' && message.includes('Live Now')) {
        return 'Go Live + Live Now';
    }
    return executionModeLabel(mode);
}

function logActionLabel(action: string) {
    switch (action) {
        case 'generate_preview':
            return 'Preview';
        case 'dry_run':
            return 'Dry Run';
        case 'go_live':
            return 'Go Live';
        case 'force_go_live':
            return 'Confirmed Go Live';
        case 'end_stream':
            return 'End Stream';
        case 'clear_pending_live_now':
            return 'Live Now Recovery';
        case 'save_destination':
            return 'Save Destination';
        case 'delete_destination':
            return 'Delete Destination';
        case 'save_settings':
            return 'Save Settings';
        default:
            return action;
    }
}

function executionStatusMessage(summary: ExecutionSummary | null) {
    if (!summary) {
        return null;
    }

    if (summary.status === 'PARTIAL') {
        if (summary.failedCount > 0 && summary.validationErrorCount > 0) {
            return 'Execution finished with issues. Some destinations failed after execution began, and others still need validation before retrying.';
        }
        if (summary.failedCount > 0) {
            return 'Execution finished with issues. Some destinations failed after execution began.';
        }
        if (summary.validationErrorCount > 0) {
            return 'Execution needs attention. Some destinations still have validation issues before they can run.';
        }
    }

    if (summary.status === 'WARNING' && summary.skippedCount > 0) {
        return 'Execution completed with warnings. One or more destinations were skipped and did not publish.';
    }

    return null;
}

function hasSecretValue(value: string) {
    return value.trim() !== '';
}

function maskSecretValue(value: string) {
    return hasSecretValue(value) ? STORED_SECRET_TOKEN : '';
}

function resolveSecretInput(value: string, storedValue: string) {
    if (value === STORED_SECRET_TOKEN) {
        return storedValue;
    }
    return value;
}

function settingsSecretCacheFrom(settings: AppSettings): SecretSettingsCache {
    return {
        testDiscordWebhookKey: settings.testDiscordWebhookKey,
        testBlueskyCredentialKey: settings.testBlueskyCredentialKey,
        testMastodonCredentialKey: settings.testMastodonCredentialKey,
    };
}

function toMaskedSettings(settings: AppSettings): AppSettings {
    return {
        ...settings,
        testDiscordWebhookKey: maskSecretValue(settings.testDiscordWebhookKey),
        testBlueskyCredentialKey: maskSecretValue(settings.testBlueskyCredentialKey),
        testMastodonCredentialKey: maskSecretValue(settings.testMastodonCredentialKey),
    };
}

function destinationSecretCacheFrom(form: DestinationFormState): DestinationSecretCache {
    return {
        discordWebhookKey: form.discordWebhookKey,
        blueskyCredentialKey: form.blueskyCredentialKey,
        mastodonCredentialKey: form.mastodonCredentialKey,
    };
}

function toMaskedDestinationForm(form: DestinationFormState): DestinationFormState {
    return {
        ...form,
        discordWebhookKey: maskSecretValue(form.discordWebhookKey),
        blueskyCredentialKey: maskSecretValue(form.blueskyCredentialKey),
        mastodonCredentialKey: maskSecretValue(form.mastodonCredentialKey),
    };
}

function App() {
    const [selectedTab, setSelectedTab] = useState<TabKey>('home');
    const [overview, setOverview] = useState<AppOverview | null>(null);
    const [bootstrapError, setBootstrapError] = useState<string | null>(null);

    const [announcement, setAnnouncement] = useState<AnnouncementInput>(initialAnnouncement);
    const [previewItems, setPreviewItems] = useState<PreviewItem[]>([]);
    const [previewError, setPreviewError] = useState<string | null>(null);
    const [previewLoading, setPreviewLoading] = useState(false);
    const [executionSummary, setExecutionSummary] = useState<ExecutionSummary | null>(null);
    const [executionError, setExecutionError] = useState<string | null>(null);
    const [executionLoading, setExecutionLoading] = useState<'dry_run' | 'go_live' | 'end_stream' | null>(null);
    const [pendingGoLiveConfirmation, setPendingGoLiveConfirmation] = useState(false);
    const [pendingLiveNowSessions, setPendingLiveNowSessions] = useState<ActiveLiveNowSession[]>([]);
    const [recoveryStatus, setRecoveryStatus] = useState<string | null>(null);
    const [recoveryError, setRecoveryError] = useState<string | null>(null);
    const [recoveryLoadingId, setRecoveryLoadingId] = useState<string | null>(null);

    const [destinations, setDestinations] = useState<DestinationInput[]>([]);
    const [destinationForm, setDestinationForm] = useState<DestinationFormState>(initialDestination);
    const [destinationSecrets, setDestinationSecrets] = useState<DestinationSecretCache>({
        discordWebhookKey: '',
        blueskyCredentialKey: '',
        mastodonCredentialKey: '',
    });
    const [destinationStatus, setDestinationStatus] = useState<string | null>(null);
    const [destinationError, setDestinationError] = useState<string | null>(null);

    const [settings, setSettings] = useState<AppSettings>(defaultSettings);
    const [settingsSecrets, setSettingsSecrets] = useState<SecretSettingsCache>({
        testDiscordWebhookKey: '',
        testBlueskyCredentialKey: '',
        testMastodonCredentialKey: '',
    });
    const [settingsStatus, setSettingsStatus] = useState<string | null>(null);
    const [settingsError, setSettingsError] = useState<string | null>(null);
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [logsError, setLogsError] = useState<string | null>(null);
    const [diagnosticsStatus, setDiagnosticsStatus] = useState<string | null>(null);

    useEffect(() => {
        getOverview()
            .then(setOverview)
            .catch((err: unknown) => {
                const message = err instanceof Error ? err.message : 'Unable to load StreamSignal overview.';
                setBootstrapError(message);
            });

        void refreshDestinations();
        void refreshSettings();
        void refreshLogs();
        void refreshPendingLiveNowSessions();
    }, []);

    async function refreshDestinations() {
        try {
            const items = await listDestinations();
            setDestinations(items);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to load destinations.';
            setDestinationError(message);
        }
    }

    async function refreshSettings() {
        try {
            const current = await getSettings();
            setSettingsSecrets(settingsSecretCacheFrom(current));
            setSettings(toMaskedSettings(current));
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to load settings.';
            setSettingsError(message);
        }
    }

    async function refreshLogs() {
        try {
            const entries = await getLogs();
            setLogs(entries);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to load logs.';
            setLogsError(message);
        }
    }

    async function refreshPendingLiveNowSessions() {
        try {
            const sessions = await listPendingLiveNowSessions();
            setPendingLiveNowSessions(sessions);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to load pending Live Now sessions.';
            setRecoveryError(message);
        }
    }

    async function onPreviewSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setPreviewLoading(true);
        setPreviewError(null);

        try {
            const preview = await generatePreview(announcement);
            setPreviewItems(preview);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to generate previews.';
            setPreviewError(message);
            setPreviewItems([]);
        } finally {
            setPreviewLoading(false);
        }
    }

    async function runExecution(mode: 'dry_run' | 'go_live') {
        setExecutionLoading(mode);
        setExecutionError(null);

        try {
            const summary = mode === 'dry_run' ? await dryRun(announcement) : await goLive(announcement);
            if (mode === 'go_live' && summary.requiresDuplicateConfirmation) {
                setPendingGoLiveConfirmation(true);
                setExecutionSummary(summary);
                return;
            }
            setPendingGoLiveConfirmation(false);
            setExecutionSummary(summary);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to run execution.';
            setExecutionError(message);
            setExecutionSummary(null);
            setPendingGoLiveConfirmation(false);
        } finally {
            setExecutionLoading(null);
        }
    }

    async function onConfirmGoLive() {
        setExecutionLoading('go_live');
        setExecutionError(null);
        try {
            const summary = await forceGoLive(announcement);
            setPendingGoLiveConfirmation(false);
            setExecutionSummary(summary);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to confirm Go Live.';
            setExecutionError(message);
        } finally {
            setExecutionLoading(null);
        }
    }

    async function onEndStream() {
        setExecutionLoading('end_stream');
        setExecutionError(null);
        setPendingGoLiveConfirmation(false);

        try {
            const summary = await endStream();
            setExecutionSummary(summary);
            await refreshPendingLiveNowSessions();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to end stream.';
            setExecutionError(message);
        } finally {
            setExecutionLoading(null);
        }
    }

    async function onClearPendingLiveNow(destinationID: string) {
        setRecoveryError(null);
        setRecoveryStatus(null);
        setRecoveryLoadingId(destinationID);

        try {
            const result = await clearPendingLiveNowSession(destinationID);
            if (result.state === 'SUCCESS') {
                setRecoveryStatus(result.message);
            } else {
                setRecoveryError(result.message);
            }
            await refreshPendingLiveNowSessions();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to clear pending Live Now session.';
            setRecoveryError(message);
        } finally {
            setRecoveryLoadingId(null);
        }
    }

    async function onDestinationSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setDestinationError(null);
        setDestinationStatus(null);

        try {
            const resolvedForm: DestinationFormState = {
                ...destinationForm,
                discordWebhookKey: resolveSecretInput(destinationForm.discordWebhookKey, destinationSecrets.discordWebhookKey),
                blueskyCredentialKey: resolveSecretInput(destinationForm.blueskyCredentialKey, destinationSecrets.blueskyCredentialKey),
                mastodonCredentialKey: resolveSecretInput(destinationForm.mastodonCredentialKey, destinationSecrets.mastodonCredentialKey),
            };
            const saved = await saveDestination(toDestinationInput(resolvedForm));
            const savedForm = toDestinationFormState(saved);
            setDestinationSecrets(destinationSecretCacheFrom(savedForm));
            setDestinationForm(toMaskedDestinationForm(savedForm));
            setDestinationStatus('Destination saved.');
            await refreshDestinations();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to save destination.';
            setDestinationError(message);
        }
    }

    async function onDeleteDestination() {
        if (!destinationForm.id) {
            return;
        }

        setDestinationError(null);
        setDestinationStatus(null);

        try {
            await deleteDestination(destinationForm.id);
            setDestinationForm(createEmptyDestinationForm(destinationForm.platform));
            setDestinationSecrets({
                discordWebhookKey: '',
                blueskyCredentialKey: '',
                mastodonCredentialKey: '',
            });
            setDestinationStatus('Destination deleted.');
            await refreshDestinations();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to delete destination.';
            setDestinationError(message);
        }
    }

    async function onSettingsSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSettingsError(null);
        setSettingsStatus(null);

        try {
            const resolvedSettings: AppSettings = {
                ...settings,
                testDiscordWebhookKey: resolveSecretInput(settings.testDiscordWebhookKey, settingsSecrets.testDiscordWebhookKey),
                testBlueskyCredentialKey: resolveSecretInput(settings.testBlueskyCredentialKey, settingsSecrets.testBlueskyCredentialKey),
                testMastodonCredentialKey: resolveSecretInput(settings.testMastodonCredentialKey, settingsSecrets.testMastodonCredentialKey),
            };
            const saved = await saveSettings(resolvedSettings);
            setSettingsSecrets(settingsSecretCacheFrom(saved));
            setSettings(toMaskedSettings(saved));
            setSettingsStatus('Settings saved.');
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to save settings.';
            setSettingsError(message);
        }
    }

    async function onCopyDiagnostics() {
        setLogsError(null);
        setDiagnosticsStatus(null);

        try {
            const diagnostics = await getDiagnostics();
            if (navigator.clipboard?.writeText) {
                await navigator.clipboard.writeText(diagnostics);
                setDiagnosticsStatus('Diagnostics copied.');
            } else {
                setLogsError('Clipboard is unavailable in this environment.');
            }
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to copy diagnostics.';
            setLogsError(message);
        }
    }

    function updateAnnouncement<K extends keyof AnnouncementInput>(field: K, value: AnnouncementInput[K]) {
        setAnnouncement((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function updateDestination<K extends keyof DestinationFormState>(field: K, value: DestinationFormState[K]) {
        setDestinationForm((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function updateSettings<K extends keyof AppSettings>(field: K, value: AppSettings[K]) {
        setSettings((current) => ({
            ...current,
            [field]: value,
        }));
    }

    return (
        <main className="app-shell">
            {settings.testModeEnabled ? (
                <div className="test-mode-banner" role="status">
                    TEST MODE ACTIVE
                </div>
            ) : null}
            <section className="hero-panel">
                <div className="hero-header">
                    <div>
                        <p className="eyebrow">{overview?.productName ?? 'StreamSignal'}</p>
                        <h1>{overview?.tagline ?? 'Send the signal. Go live everywhere.'}</h1>
                    </div>
                    <span className="status-pill">Milestone 7</span>
                </div>
                <p className="hero-copy">
                    {bootstrapError
                        ? bootstrapError
                        : overview?.currentPhase ?? 'Configuration screens are the next layer of the app shell.'}
                </p>
            </section>

            <section className="workspace-grid">
                <nav className="tab-bar" aria-label="Primary navigation">
                    <button className={selectedTab === 'home' ? 'tab-button active' : 'tab-button'} onClick={() => setSelectedTab('home')}>
                        Home
                    </button>
                    <button
                        className={selectedTab === 'destinations' ? 'tab-button active' : 'tab-button'}
                        onClick={() => setSelectedTab('destinations')}
                    >
                        Destinations
                    </button>
                    <button
                        className={selectedTab === 'settings' ? 'tab-button active' : 'tab-button'}
                        onClick={() => setSelectedTab('settings')}
                    >
                        Settings
                    </button>
                    <button className={selectedTab === 'logs' ? 'tab-button active' : 'tab-button'} onClick={() => setSelectedTab('logs')}>
                        Logs
                    </button>
                </nav>

                {selectedTab === 'home' ? (
                    <>
                        <article className="panel">
                            <div className="panel-header">
                                <h2>Home</h2>
                                <span>Preview workflow</span>
                            </div>

                            <form className="announcement-form" onSubmit={onPreviewSubmit}>
                                <label className="field">
                                    <span>Stream Title</span>
                                    <input
                                        aria-label="Stream Title"
                                        value={announcement.streamTitle}
                                        onChange={(event) => updateAnnouncement('streamTitle', event.target.value)}
                                        placeholder="Late Night Variety"
                                    />
                                </label>

                                <label className="field">
                                    <span>Stream URL</span>
                                    <input
                                        aria-label="Stream URL"
                                        value={announcement.streamURL}
                                        onChange={(event) => updateAnnouncement('streamURL', event.target.value)}
                                        placeholder="https://example.com/live"
                                    />
                                </label>

                                <label className="field">
                                    <span>Category / Game</span>
                                    <input
                                        aria-label="Category / Game"
                                        value={announcement.category}
                                        onChange={(event) => updateAnnouncement('category', event.target.value)}
                                        placeholder="Music, FFXIV, Variety..."
                                    />
                                </label>

                                <label className="field">
                                    <span>Optional Message</span>
                                    <textarea
                                        aria-label="Optional Message"
                                        value={announcement.message}
                                        onChange={(event) => updateAnnouncement('message', event.target.value)}
                                        placeholder="What are we getting into tonight?"
                                        rows={4}
                                    />
                                </label>

                                <label className="field">
                                    <span>Hashtags</span>
                                    <input
                                        aria-label="Hashtags"
                                        value={announcement.hashtags}
                                        onChange={(event) => updateAnnouncement('hashtags', event.target.value)}
                                        placeholder="#vtuber #music"
                                    />
                                </label>

                                <div className="form-actions">
                                    <button type="submit" className="primary-button" disabled={previewLoading}>
                                        {previewLoading ? 'Generating...' : 'Generate Preview'}
                                    </button>
                                    <button
                                        type="button"
                                        className="ghost-button"
                                        onClick={() => {
                                            setAnnouncement(initialAnnouncement);
                                            setPreviewError(null);
                                            setPreviewItems([]);
                                        }}
                                    >
                                        Reset
                                    </button>
                                    <button
                                        type="button"
                                        className="ghost-button"
                                        onClick={() => void runExecution('dry_run')}
                                        disabled={executionLoading !== null}
                                    >
                                        {executionLoading === 'dry_run' ? 'Running Dry Run...' : 'Dry Run'}
                                    </button>
                                    <button
                                        type="button"
                                        className="primary-button"
                                        onClick={() => void runExecution('go_live')}
                                        disabled={executionLoading !== null}
                                    >
                                        {executionLoading === 'go_live' ? 'Running Go Live...' : 'Go Live'}
                                    </button>
                                    <button
                                        type="button"
                                        className="ghost-button"
                                        onClick={() => void onEndStream()}
                                        disabled={executionLoading !== null}
                                    >
                                        {executionLoading === 'end_stream' ? 'Ending Stream...' : 'End Stream'}
                                    </button>
                                </div>
                            </form>
                        </article>

                        <article className="panel">
                            <div className="panel-header">
                                <h2>Preview Panel</h2>
                                <span>{previewItems.length} destinations</span>
                            </div>

                            {previewError ? <p className="error-banner">{previewError}</p> : null}

                            {previewItems.length === 0 ? (
                                <div className="empty-state">
                                    <h3>No previews yet</h3>
                                    <p>Generate a preview to see per-destination content, character counts, and validation state.</p>
                                </div>
                            ) : (
                                <div className="preview-list">
                                    {previewItems.map((item) => (
                                        <section key={item.destinationID} className="preview-card">
                                            <div className="preview-topline">
                                                <div>
                                                    <h3>{item.destinationName}</h3>
                                                    <p className="preview-platform">{item.platform}</p>
                                                </div>
                                                <span className={`preview-status status-${item.validationState.toLowerCase()}`}>
                                                    {item.validationState}
                                                </span>
                                            </div>

                                            <pre className="preview-content">{item.content || '(empty content)'}</pre>

                                            <div className="preview-meta">
                                                <span>{item.characterCount} characters</span>
                                                <span>{item.validationNotes.length} notes</span>
                                            </div>

                                            {item.validationNotes.length > 0 ? (
                                                <ul className="note-list">
                                                    {item.validationNotes.map((note) => (
                                                        <li key={note}>{note}</li>
                                                    ))}
                                                </ul>
                                            ) : (
                                                <p className="success-copy">This preview is ready from a validation standpoint.</p>
                                            )}
                                        </section>
                                    ))}
                                </div>
                            )}
                        </article>

                        <article className="panel panel-wide">
                            <div className="panel-header">
                                <h2>Execution Results</h2>
                                <span className={executionSummary ? `status-pill status-${executionSummary.status.toLowerCase()}` : 'status-pill'}>
                                    {executionSummary?.status ?? executionSummary?.mode ?? 'idle'}
                                </span>
                            </div>
                            {executionError ? <p className="error-banner">{executionError}</p> : null}
                            {executionStatusMessage(executionSummary) ? (
                                <div className={executionSummary?.status === 'PARTIAL' ? 'error-banner' : 'warning-banner'}>
                                    <p>{executionStatusMessage(executionSummary)}</p>
                                </div>
                            ) : null}
                            {executionSummary?.testModeActive ? (
                                <div className="warning-banner">
                                    <p>
                                        {executionSummary.mode === 'dry_run'
                                            ? 'Test Mode is active. Any live execution would be routed to test targets instead of production destinations.'
                                            : executionSummary.mode === 'end_stream'
                                              ? 'Test Mode routing was active. Any optional end-stream posts used test targets instead of production destinations.'
                                              : 'Test Mode routing was active. Production destinations were not used for this execution.'}
                                    </p>
                                </div>
                            ) : null}
                            {pendingGoLiveConfirmation ? (
                                <div className="warning-banner">
                                    <p>{executionSummary?.duplicateWarningMessage}</p>
                                    <div className="form-actions">
                                        <button
                                            type="button"
                                            className="primary-button"
                                            onClick={() => void onConfirmGoLive()}
                                            disabled={executionLoading !== null}
                                        >
                                            {executionLoading === 'go_live' ? 'Confirming...' : 'Confirm Go Live'}
                                        </button>
                                    </div>
                                </div>
                            ) : null}
                            {!executionSummary ? (
                                <div className="empty-state">
                                    <h3>No execution results yet</h3>
                                    <p>Run Dry Run, Go Live, or End Stream to see per-destination execution results.</p>
                                </div>
                            ) : (
                                <>
                                    <div className="execution-summary-grid">
                                        <section className="mini-panel">
                                            <h3>Total</h3>
                                            <p className="summary-metric">{executionSummary.totalCount}</p>
                                        </section>
                                        <section className="mini-panel">
                                            <h3>Success</h3>
                                            <p className="summary-metric">{executionSummary.successCount}</p>
                                        </section>
                                        <section className="mini-panel">
                                            <h3>Failed</h3>
                                            <p className="summary-metric">{executionSummary.failedCount}</p>
                                        </section>
                                        <section className="mini-panel">
                                            <h3>Validation</h3>
                                            <p className="summary-metric">{executionSummary.validationErrorCount}</p>
                                        </section>
                                        <section className="mini-panel">
                                            <h3>Skipped</h3>
                                            <p className="summary-metric">{executionSummary.skippedCount}</p>
                                        </section>
                                    </div>

                                    {executionSummary.results.length === 0 ? (
                                        <div className="empty-state">
                                            <h3>No per-destination rows yet</h3>
                                            <p>Execution warnings can appear before any live publish attempt is made.</p>
                                        </div>
                                    ) : (
                                        <div className="logs-list">
                                            {executionSummary.results.map((result) => (
                                                <section className="log-row" key={`${result.destinationID}-${result.state}`}>
                                                    <div className="log-summary">
                                                        <strong>{result.destinationName}</strong>
                                                        <span>{result.state}</span>
                                                    </div>
                                                    <p className="log-context">
                                                        {executionResultKind(executionSummary.mode, result.message)} · {result.platform}
                                                    </p>
                                                    <p>{result.message}</p>
                                                    {result.content ? <p className="log-content-preview">{result.content}</p> : null}
                                                </section>
                                            ))}
                                        </div>
                                    )}
                                </>
                            )}
                        </article>
                    </>
                ) : null}

                {selectedTab === 'destinations' ? (
                    <>
                        <article className="panel">
                            <div className="panel-header">
                                <h2>Destinations</h2>
                                <span>{destinations.length} configured</span>
                            </div>
                            {destinationError ? <p className="error-banner">{destinationError}</p> : null}
                            {destinationStatus ? <p className="success-banner">{destinationStatus}</p> : null}

                            <div className="destination-list">
                                {destinations.length === 0 ? (
                                    <div className="empty-state">
                                        <h3>No destinations yet</h3>
                                        <p>Create a destination to start building platform-specific previews and go-live targets.</p>
                                    </div>
                                ) : (
                                    destinations.map((item) => (
                                        <button
                                            key={item.id}
                                            className={destinationForm.id === item.id ? 'destination-item active' : 'destination-item'}
                                            onClick={() => {
                                                const form = toDestinationFormState(item);
                                                setDestinationSecrets(destinationSecretCacheFrom(form));
                                                setDestinationForm(toMaskedDestinationForm(form));
                                            }}
                                        >
                                            <strong>{item.name}</strong>
                                            <span>{item.platform}</span>
                                        </button>
                                    ))
                                )}
                            </div>
                        </article>

                        <article className="panel">
                            <div className="panel-header">
                                <h2>{destinationForm.id ? 'Edit Destination' : 'New Destination'}</h2>
                                <span>CRUD-ready</span>
                            </div>

                            <form className="announcement-form" onSubmit={onDestinationSubmit}>
                                <label className="field">
                                    <span>Platform</span>
                                    <select
                                        aria-label="Platform"
                                        value={destinationForm.platform}
                                        onChange={(event) => {
                                            setDestinationSecrets({
                                                discordWebhookKey: '',
                                                blueskyCredentialKey: '',
                                                mastodonCredentialKey: '',
                                            });
                                            setDestinationForm((current) => ({
                                                ...createEmptyDestinationForm(event.target.value as DestinationInput['platform']),
                                                id: current.id,
                                                name: current.name,
                                                enabled: current.enabled,
                                                createdAt: current.createdAt,
                                                updatedAt: current.updatedAt,
                                            }));
                                        }}
                                    >
                                        <option value="discord">Discord</option>
                                        <option value="bluesky">Bluesky</option>
                                        <option value="mastodon">Mastodon</option>
                                    </select>
                                </label>

                                <label className="field">
                                    <span>Friendly Name</span>
                                    <input
                                        aria-label="Friendly Name"
                                        value={destinationForm.name}
                                        onChange={(event) => updateDestination('name', event.target.value)}
                                        placeholder="Main Discord"
                                    />
                                </label>

                                <label className="field checkbox-field">
                                    <input
                                        aria-label="Enabled"
                                        type="checkbox"
                                        checked={destinationForm.enabled}
                                        onChange={(event) => updateDestination('enabled', event.target.checked)}
                                    />
                                    <span>Enabled</span>
                                </label>

                                <label className="field">
                                    <span>{destinationForm.platform === 'bluesky' ? 'Post Template' : 'Template'}</span>
                                    <textarea
                                        aria-label={destinationForm.platform === 'bluesky' ? 'Post Template' : 'Template'}
                                        value={destinationForm.template}
                                        onChange={(event) => updateDestination('template', event.target.value)}
                                        placeholder="{{stream_title}}"
                                        rows={5}
                                    />
                                </label>

                                {destinationForm.platform === 'discord' ? (
                                    <>
                                        <label className="field">
                                            <span>Server Name</span>
                                            <input
                                                aria-label="Server Name"
                                                value={destinationForm.discordServerName}
                                                onChange={(event) => updateDestination('discordServerName', event.target.value)}
                                                placeholder="My Discord Server"
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Channel Name</span>
                                            <input
                                                aria-label="Channel Name"
                                                value={destinationForm.discordChannelName}
                                                onChange={(event) => updateDestination('discordChannelName', event.target.value)}
                                                placeholder="go-live"
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Webhook Key</span>
                                            <input
                                                aria-label="Webhook Key"
                                                value={destinationForm.discordWebhookKey}
                                                onChange={(event) => updateDestination('discordWebhookKey', event.target.value)}
                                                placeholder="discord/main"
                                            />
                                        </label>
                                    </>
                                ) : null}

                                {destinationForm.platform === 'bluesky' ? (
                                    <>
                                        <label className="field">
                                            <span>Account Identifier</span>
                                            <input
                                                aria-label="Account Identifier"
                                                value={destinationForm.blueskyAccountIdentifier}
                                                onChange={(event) => updateDestination('blueskyAccountIdentifier', event.target.value)}
                                                placeholder="don.test"
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Credential Key</span>
                                            <input
                                                aria-label="Credential Key"
                                                value={destinationForm.blueskyCredentialKey}
                                                onChange={(event) => updateDestination('blueskyCredentialKey', event.target.value)}
                                                placeholder="bluesky/main"
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Live Status Template</span>
                                            <textarea
                                                aria-label="Live Status Template"
                                                value={destinationForm.blueskyLiveStatusTemplate}
                                                onChange={(event) => updateDestination('blueskyLiveStatusTemplate', event.target.value)}
                                                placeholder="LIVE {{stream_title}}"
                                                rows={4}
                                            />
                                        </label>
                                    </>
                                ) : null}

                                {destinationForm.platform === 'mastodon' ? (
                                    <>
                                        <label className="field">
                                            <span>Account Identifier</span>
                                            <input
                                                aria-label="Account Identifier"
                                                value={destinationForm.mastodonAccountIdentifier}
                                                onChange={(event) => updateDestination('mastodonAccountIdentifier', event.target.value)}
                                                placeholder="@don@example.social"
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Instance URL</span>
                                            <input
                                                aria-label="Instance URL"
                                                value={destinationForm.mastodonInstanceURL}
                                                onChange={(event) => updateDestination('mastodonInstanceURL', event.target.value)}
                                                placeholder="https://mastodon.social"
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Credential Key</span>
                                            <input
                                                aria-label="Credential Key"
                                                value={destinationForm.mastodonCredentialKey}
                                                onChange={(event) => updateDestination('mastodonCredentialKey', event.target.value)}
                                                placeholder="mastodon/main"
                                            />
                                        </label>
                                    </>
                                ) : null}

                                <div className="form-actions">
                                    <button type="submit" className="primary-button">
                                        Save Destination
                                    </button>
                                    <button
                                        type="button"
                                        className="ghost-button"
                                        onClick={() => {
                                            setDestinationSecrets({
                                                discordWebhookKey: '',
                                                blueskyCredentialKey: '',
                                                mastodonCredentialKey: '',
                                            });
                                            setDestinationForm(createEmptyDestinationForm(destinationForm.platform));
                                        }}
                                    >
                                        New
                                    </button>
                                    <button
                                        type="button"
                                        className="danger-button"
                                        onClick={() => void onDeleteDestination()}
                                        disabled={!destinationForm.id}
                                    >
                                        Delete
                                    </button>
                                </div>
                            </form>
                        </article>
                    </>
                ) : null}

                {selectedTab === 'settings' ? (
                    <article className="panel panel-wide">
                        <div className="panel-header">
                            <h2>Settings</h2>
                            <span>App defaults and safety rails</span>
                        </div>
                        {settingsError ? <p className="error-banner">{settingsError}</p> : null}
                        {settingsStatus ? <p className="success-banner">{settingsStatus}</p> : null}

                        <form className="announcement-form settings-grid" onSubmit={onSettingsSubmit}>
                            <label className="field checkbox-field">
                                <input
                                    aria-label="Test Mode Enabled"
                                    type="checkbox"
                                    checked={settings.testModeEnabled}
                                    onChange={(event) => updateSettings('testModeEnabled', event.target.checked)}
                                />
                                <span>Test Mode Enabled</span>
                            </label>

                            <label className="field">
                                <span>Test Discord Webhook Key</span>
                                <input
                                    aria-label="Test Discord Webhook Key"
                                    value={settings.testDiscordWebhookKey}
                                    onChange={(event) => updateSettings('testDiscordWebhookKey', event.target.value)}
                                    placeholder="discord/test"
                                />
                            </label>

                            <label className="field">
                                <span>Test Bluesky Account Identifier</span>
                                <input
                                    aria-label="Test Bluesky Account Identifier"
                                    value={settings.testBlueskyAccountIdentifier}
                                    onChange={(event) => updateSettings('testBlueskyAccountIdentifier', event.target.value)}
                                    placeholder="don.test"
                                />
                            </label>

                            <label className="field">
                                <span>Test Bluesky Credential Key</span>
                                <input
                                    aria-label="Test Bluesky Credential Key"
                                    value={settings.testBlueskyCredentialKey}
                                    onChange={(event) => updateSettings('testBlueskyCredentialKey', event.target.value)}
                                    placeholder="bluesky/test"
                                />
                            </label>

                            <label className="field">
                                <span>Test Mastodon Credential Key</span>
                                <input
                                    aria-label="Test Mastodon Credential Key"
                                    value={settings.testMastodonCredentialKey}
                                    onChange={(event) => updateSettings('testMastodonCredentialKey', event.target.value)}
                                    placeholder="mastodon/test"
                                />
                            </label>

                            <label className="field">
                                <span>Test Mastodon Instance URL</span>
                                <input
                                    aria-label="Test Mastodon Instance URL"
                                    value={settings.testMastodonInstanceURL}
                                    onChange={(event) => updateSettings('testMastodonInstanceURL', event.target.value)}
                                    placeholder="https://mastodon.test"
                                />
                            </label>

                            <label className="field checkbox-field">
                                <input
                                    aria-label="Duplicate Protection Enabled"
                                    type="checkbox"
                                    checked={settings.duplicateProtectionEnabled}
                                    onChange={(event) => updateSettings('duplicateProtectionEnabled', event.target.checked)}
                                />
                                <span>Duplicate Protection Enabled</span>
                            </label>

                            <label className="field checkbox-field">
                                <input
                                    aria-label="End Stream Post Enabled"
                                    type="checkbox"
                                    checked={settings.endStreamPostEnabled}
                                    onChange={(event) => updateSettings('endStreamPostEnabled', event.target.checked)}
                                />
                                <span>End Stream Post Enabled</span>
                            </label>

                            <label className="field">
                                <span>Default Stream URL</span>
                                <input
                                    aria-label="Default Stream URL"
                                    value={settings.defaultStreamURL}
                                    onChange={(event) => updateSettings('defaultStreamURL', event.target.value)}
                                    placeholder="https://example.com/live"
                                />
                            </label>

                            <label className="field">
                                <span>Default Hashtags</span>
                                <input
                                    aria-label="Default Hashtags"
                                    value={settings.defaultHashtags}
                                    onChange={(event) => updateSettings('defaultHashtags', event.target.value)}
                                    placeholder="#vtuber #music"
                                />
                            </label>

                            <label className="field">
                                <span>Duplicate Window (Minutes)</span>
                                <input
                                    aria-label="Duplicate Window (Minutes)"
                                    type="number"
                                    min={1}
                                    value={settings.duplicateWindowMinutes}
                                    onChange={(event) => updateSettings('duplicateWindowMinutes', Number(event.target.value))}
                                />
                            </label>

                            <label className="field panel-wide">
                                <span>End Stream Template</span>
                                <textarea
                                    aria-label="End Stream Template"
                                    value={settings.endStreamTemplate}
                                    onChange={(event) => updateSettings('endStreamTemplate', event.target.value)}
                                    placeholder="Thanks for hanging out!"
                                    rows={4}
                                />
                            </label>

                            <div className="form-actions panel-wide">
                                <button type="submit" className="primary-button">
                                    Save Settings
                                </button>
                            </div>
                        </form>

                        <section className="mini-panel">
                            <h3>Live Now Recovery</h3>
                            <p>Use this if Stream Signal closed before Bluesky Live Now was cleared, or if a pending session needs to be cleaned up manually.</p>
                            {recoveryError ? <p className="error-banner">{recoveryError}</p> : null}
                            {recoveryStatus ? <p className="success-banner">{recoveryStatus}</p> : null}

                            {pendingLiveNowSessions.length === 0 ? (
                                <p>No pending Live Now sessions.</p>
                            ) : (
                                <div className="logs-list">
                                    {pendingLiveNowSessions.map((session) => (
                                        <section className="log-row" key={session.destinationID}>
                                            <div className="log-summary">
                                                <strong>{session.destinationName}</strong>
                                                <span>{session.platform}</span>
                                            </div>
                                            <p>{session.streamTitle || 'Untitled stream'}</p>
                                            <p>{session.streamURL}</p>
                                            <button
                                                type="button"
                                                className="ghost-button"
                                                onClick={() => void onClearPendingLiveNow(session.destinationID)}
                                                disabled={recoveryLoadingId !== null}
                                            >
                                                {recoveryLoadingId === session.destinationID ? 'Recovering Live Now...' : 'Recover and Clear Live Now'}
                                            </button>
                                        </section>
                                    ))}
                                </div>
                            )}
                        </section>
                    </article>
                ) : null}

                {selectedTab === 'logs' ? (
                    <article className="panel panel-wide">
                        <div className="panel-header">
                            <h2>Logs</h2>
                            <span>{logs.length} recent entries</span>
                        </div>
                        {logsError ? <p className="error-banner">{logsError}</p> : null}
                        {diagnosticsStatus ? <p className="success-banner">{diagnosticsStatus}</p> : null}

                        <div className="form-actions logs-actions">
                            <button type="button" className="ghost-button" onClick={() => void refreshLogs()}>
                                Refresh Logs
                            </button>
                            <button type="button" className="primary-button" onClick={() => void onCopyDiagnostics()}>
                                Copy Diagnostics
                            </button>
                        </div>

                        {logs.length === 0 ? (
                            <div className="empty-state">
                                <h3>No logs yet</h3>
                                <p>Workflow activity and validation outcomes will appear here as you use the app.</p>
                            </div>
                        ) : (
                            <div className="logs-list">
                                {logs.map((entry, index) => (
                                    <section className="log-row" key={`${entry.timestamp}-${entry.action}-${index}`}>
                                        <div className="log-summary">
                                            <strong>{entry.destination}</strong>
                                            <span>{entry.status}</span>
                                        </div>
                                        <p className="log-context">{logActionLabel(entry.action)}</p>
                                        <p>{entry.message}</p>
                                        <time dateTime={entry.timestamp}>{entry.timestamp}</time>
                                    </section>
                                ))}
                            </div>
                        )}
                    </article>
                ) : null}

                {selectedTab === 'home' && overview ? (
                    <article className="panel panel-wide">
                        <div className="panel-header">
                            <h2>Milestone Radar</h2>
                            <span>Current map</span>
                        </div>
                        <div className="milestone-grid">
                            <div className="mini-panel">
                                <h3>Now</h3>
                                <ul className="list">
                                    {overview.nextActions.map((item) => (
                                        <li key={item}>{item}</li>
                                    ))}
                                </ul>
                            </div>
                            <div className="mini-panel">
                                <h3>Guardrails</h3>
                                <ul className="list">
                                    {overview.highlights.map((item) => (
                                        <li key={item}>{item}</li>
                                    ))}
                                </ul>
                            </div>
                        </div>
                    </article>
                ) : null}
            </section>
        </main>
    );
}

export default App;
