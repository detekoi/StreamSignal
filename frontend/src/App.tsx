import { FormEvent, useEffect, useRef, useState } from 'react';
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
    listPendingLiveNowSessions,
    getSettings,
    listDestinations,
    saveDestination,
    saveSettings,
    testDestinationConnection,
} from './lib/api/streamsignal';
import type { CredentialCheckResult } from './types/credential-check';
import {
    createEmptyDestinationForm,
    defaultTemplateForPlatform,
    TEMPLATE_VARIABLES,
    toDestinationFormState,
    toDestinationInput,
    usedTemplateVariables,
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

type GuidedSetupStep = {
    title: string;
    detail: string;
};

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

function credentialSetupLabel(platform: DestinationInput['platform']) {
    switch (platform) {
        case 'discord':
            return 'Discord webhook';
        case 'bluesky':
            return 'Bluesky app password';
        case 'mastodon':
            return 'Mastodon access token';
        default:
            return 'credential';
    }
}

function guidedSetupSteps(platform: DestinationInput['platform']): GuidedSetupStep[] {
    switch (platform) {
        case 'discord':
            return [
                {
                    title: 'Step 1: Create a channel webhook',
                    detail: 'In Discord, open Server Settings > Integrations > Webhooks, create a webhook for your go-live channel, and copy the webhook URL.',
                },
                {
                    title: 'Step 2: Paste the webhook URL below',
                    detail: 'Use the full Discord webhook URL. StreamSignal stores it in Windows Credential Manager, not plain text app storage.',
                },
                {
                    title: 'Step 3: Test before saving',
                    detail: 'Use Test Connection to confirm the webhook responds before you save this destination.',
                },
            ];
        case 'bluesky':
            return [
                {
                    title: 'Step 1: Create an app password',
                    detail: 'In Bluesky, open Settings > Privacy and Security > App Passwords, create one for StreamSignal, and copy it once.',
                },
                {
                    title: 'Step 2: Enter your handle and app password',
                    detail: 'Use the Bluesky handle or account identifier for the account that should post and set Live Now.',
                },
                {
                    title: 'Step 3: Test login before saving',
                    detail: 'Test Connection signs in with the app password so you know the account can be reached before you depend on it.',
                },
            ];
        case 'mastodon':
            return [
                {
                    title: 'Step 1: Create a posting token',
                    detail: 'In your Mastodon account settings, create an access token with permission to post statuses, then copy the token and your instance URL.',
                },
                {
                    title: 'Step 2: Paste the instance URL and token',
                    detail: 'Use the full instance URL, like https://mastodon.social, plus the token that belongs to the account you want StreamSignal to use.',
                },
                {
                    title: 'Step 3: Verify the token before saving',
                    detail: 'Test Connection checks the token against your Mastodon account so invalid or expired tokens fail early.',
                },
            ];
        default:
            return [];
    }
}

function destinationTemplateLabel(platform: DestinationInput['platform']) {
    return platform === 'bluesky' ? 'Post Template' : 'Template';
}

function App() {
    const [selectedTab, setSelectedTab] = useState<TabKey>('home');

    const [announcement, setAnnouncement] = useState<AnnouncementInput>(initialAnnouncement);
    const [selectedDestinationIDs, setSelectedDestinationIDs] = useState<string[]>([]);
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
    const [destinationConnectionResult, setDestinationConnectionResult] = useState<CredentialCheckResult | null>(null);
    const [destinationConnectionLoading, setDestinationConnectionLoading] = useState(false);
    const [destinationHelperStatus, setDestinationHelperStatus] = useState<string | null>(null);
    const [showGuidedSetup, setShowGuidedSetup] = useState(false);

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
    const initializedSelection = useRef(false);

    useEffect(() => {
        void refreshDestinations();
        void refreshSettings();
        void refreshLogs();
        void refreshPendingLiveNowSessions();
    }, []);

    useEffect(() => {
        const availableIDs = destinations.map((destination) => destination.id);
        if (availableIDs.length > 0 && !initializedSelection.current) {
            initializedSelection.current = true;
            setSelectedDestinationIDs(availableIDs);
            return;
        }

        setSelectedDestinationIDs((current) => current.filter((id) => availableIDs.includes(id)));
    }, [destinations]);

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
            if (destinations.length > 0 && selectedDestinationIDs.length === 0) {
                setPreviewError('Select at least one destination for this session before generating previews.');
                setPreviewItems([]);
                return;
            }

            const preview = await generatePreview(announcement, selectedDestinationIDs);
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
            if (destinations.length > 0 && selectedDestinationIDs.length === 0) {
                setExecutionError('Select at least one destination for this session before running this action.');
                setExecutionSummary(null);
                setPendingGoLiveConfirmation(false);
                return;
            }

            const summary = mode === 'dry_run' ? await dryRun(announcement, selectedDestinationIDs) : await goLive(announcement, selectedDestinationIDs);
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
            const summary = await forceGoLive(announcement, selectedDestinationIDs);
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
            setSelectedDestinationIDs((current) => (current.includes(saved.id) ? current : [...current, saved.id]));
            setDestinationStatus('Destination saved.');
            await refreshDestinations();
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to save destination.';
            setDestinationError(message);
        }
    }

    async function onTestDestinationConnection() {
        setDestinationError(null);
        setDestinationStatus(null);
        setDestinationConnectionResult(null);
        setDestinationConnectionLoading(true);

        try {
            const resolvedForm: DestinationFormState = {
                ...destinationForm,
                discordWebhookKey: resolveSecretInput(destinationForm.discordWebhookKey, destinationSecrets.discordWebhookKey),
                blueskyCredentialKey: resolveSecretInput(destinationForm.blueskyCredentialKey, destinationSecrets.blueskyCredentialKey),
                mastodonCredentialKey: resolveSecretInput(destinationForm.mastodonCredentialKey, destinationSecrets.mastodonCredentialKey),
            };
            const result = await testDestinationConnection(toDestinationInput(resolvedForm));
            setDestinationConnectionResult(result);
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to test destination connection.';
            setDestinationError(message);
        } finally {
            setDestinationConnectionLoading(false);
        }
    }

    async function onDeleteDestination() {
        if (!destinationForm.id) {
            return;
        }

        setDestinationError(null);
        setDestinationStatus(null);

        try {
            const deletedID = destinationForm.id;
            await deleteDestination(destinationForm.id);
            setDestinationForm(createEmptyDestinationForm(destinationForm.platform));
            setDestinationSecrets({
                discordWebhookKey: '',
                blueskyCredentialKey: '',
                mastodonCredentialKey: '',
            });
            setSelectedDestinationIDs((current) => current.filter((id) => id !== deletedID));
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
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationForm((current) => ({
            ...current,
            [field]: value,
        }));
    }

    async function onCopyTemplateVariable(variable: string) {
        setDestinationHelperStatus(null);
        try {
            if (navigator.clipboard?.writeText) {
                await navigator.clipboard.writeText(variable);
                setDestinationHelperStatus(`Copied ${variable}`);
                return;
            }
            setDestinationHelperStatus('Clipboard is unavailable in this environment.');
        } catch (err: unknown) {
            const message = err instanceof Error ? err.message : 'Unable to copy template variable.';
            setDestinationHelperStatus(message);
        }
    }

    function updateSettings<K extends keyof AppSettings>(field: K, value: AppSettings[K]) {
        setSettings((current) => ({
            ...current,
            [field]: value,
        }));
    }

    function toggleSessionDestination(destinationID: string) {
        setSelectedDestinationIDs((current) =>
            current.includes(destinationID) ? current.filter((id) => id !== destinationID) : [...current, destinationID],
        );
    }

    const configuredDestinationCount = destinations.length;
    const selectedDestinationCount = selectedDestinationIDs.length;

    return (
        <main className="app-shell">
            {settings.testModeEnabled ? (
                <div className="test-mode-banner" role="status">
                    TEST MODE ACTIVE
                </div>
            ) : null}
            <div className="app-frame">
                <header className="panel app-header">
                    <div className="app-header-copy">
                        <h1>StreamSignal</h1>
                    </div>
                    <nav className="tab-bar" aria-label="Primary navigation">
                            <button
                                aria-label="Home"
                                className={selectedTab === 'home' ? 'tab-button active' : 'tab-button'}
                                onClick={() => setSelectedTab('home')}
                            >
                                Home
                            </button>
                            <button
                                aria-label="Destinations"
                                className={selectedTab === 'destinations' ? 'tab-button active' : 'tab-button'}
                                onClick={() => setSelectedTab('destinations')}
                            >
                                Destinations
                            </button>
                            <button
                                aria-label="Settings"
                                className={selectedTab === 'settings' ? 'tab-button active' : 'tab-button'}
                                onClick={() => setSelectedTab('settings')}
                            >
                                Settings
                            </button>
                            <button
                                aria-label="Logs"
                                className={selectedTab === 'logs' ? 'tab-button active' : 'tab-button'}
                                onClick={() => setSelectedTab('logs')}
                            >
                                Logs
                            </button>
                    </nav>
                </header>

                <section className="content-stack">
                        {selectedTab === 'home' ? (
                            <>
                                <article className="panel home-session-panel">
                                    <div className="panel-header">
                                        <div>
                                            <h2>Live Session</h2>
                                            <p>Set up the announcement, validate it, then publish when you are ready.</p>
                                        </div>
                                        <div className="home-status-inline">
                                            {settings.testModeEnabled ? <span className="status-pill">Test Mode</span> : null}
                                            {pendingLiveNowSessions.length > 0 ? (
                                                <span className="status-pill">{pendingLiveNowSessions.length} recovery pending</span>
                                            ) : null}
                                        </div>
                                    </div>

                                    <form className="announcement-form" onSubmit={onPreviewSubmit}>
                                        <div className="form-grid">
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
                                                <span>Hashtags</span>
                                                <input
                                                    aria-label="Hashtags"
                                                    value={announcement.hashtags}
                                                    onChange={(event) => updateAnnouncement('hashtags', event.target.value)}
                                                    placeholder="#vtuber #music"
                                                />
                                            </label>
                                        </div>

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

                                        <section className="session-destination-panel">
                                            <div className="session-destination-header">
                                                <div>
                                                    <h3>Send To</h3>
                                                    <p>
                                                        {configuredDestinationCount === 0
                                                            ? 'Add destinations in the Destinations tab first.'
                                                            : `${selectedDestinationCount} of ${configuredDestinationCount} selected for this session.`}
                                                        {settings.testModeEnabled && configuredDestinationCount > 0 ? ' Test Mode will reroute live posting to your test credentials.' : ''}
                                                        {pendingLiveNowSessions.length > 0 ? ` ${pendingLiveNowSessions.length} Live Now recovery item${pendingLiveNowSessions.length === 1 ? ' is' : 's are'} still pending.` : ''}
                                                    </p>
                                                </div>
                                                {configuredDestinationCount > 0 ? (
                                                    <div className="session-destination-actions">
                                                        <button
                                                            type="button"
                                                            className="ghost-button"
                                                            onClick={() => setSelectedDestinationIDs(destinations.map((destination) => destination.id))}
                                                        >
                                                            All
                                                        </button>
                                                        <button
                                                            type="button"
                                                            className="ghost-button"
                                                            onClick={() => setSelectedDestinationIDs([])}
                                                        >
                                                            None
                                                        </button>
                                                    </div>
                                                ) : null}
                                            </div>

                                            {configuredDestinationCount > 0 ? (
                                                <div className="session-destination-list">
                                                    {destinations.map((destination) => (
                                                        <label key={destination.id} className="session-destination-option">
                                                            <input
                                                                type="checkbox"
                                                                aria-label={destination.name}
                                                                checked={selectedDestinationIDs.includes(destination.id)}
                                                                onChange={() => toggleSessionDestination(destination.id)}
                                                            />
                                                            <span className="session-destination-copy">
                                                                <strong>{destination.name}</strong>
                                                                <small>{destination.platform}</small>
                                                            </span>
                                                        </label>
                                                    ))}
                                                </div>
                                            ) : null}
                                        </section>

                                        <div className="workflow-actions">
                                            <div className="workflow-action-group">
                                                <span className="workflow-label">Prepare</span>
                                                <div className="action-cluster-main">
                                                    <button type="submit" className="primary-button" disabled={previewLoading}>
                                                        {previewLoading ? 'Generating...' : 'Generate Preview'}
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
                                                        className="ghost-button"
                                                        onClick={() => {
                                                            setAnnouncement(initialAnnouncement);
                                                            setPreviewError(null);
                                                            setPreviewItems([]);
                                                        }}
                                                    >
                                                        Reset
                                                    </button>
                                                </div>
                                            </div>
                                            <div className="workflow-action-group workflow-action-group-live">
                                                <span className="workflow-label">Commit</span>
                                                <div className="action-cluster-secondary">
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
                                            </div>
                                        </div>
                                    </form>
                                </article>

                                <article className="panel">
                                        <div className="panel-header">
                                            <div>
                                                <h2>Preview Panel</h2>
                                                <p>See the exact message each selected destination will receive.</p>
                                            </div>
                                            <span>{previewItems.length} destinations</span>
                                        </div>

                                    {previewError ? <p className="error-banner">{previewError}</p> : null}

                                    {previewItems.length === 0 ? (
                                        <div className="empty-state">
                                            <h3>No previews yet</h3>
                                            <p>
                                                {destinations.length === 0
                                                    ? 'Add at least one destination first, then generate a preview to see per-destination content.'
                                                    : selectedDestinationIDs.length === 0
                                                      ? 'Select at least one destination for this session, then generate a preview to see per-destination content.'
                                                      : 'Generate a preview to see per-destination content, character counts, and validation state.'}
                                            </p>
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

                                <article className="panel">
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
                                <article className="panel panel-wide">
                                    <div className="panel-header">
                                        <div>
                                            <h2>Destinations</h2>
                                            <p>Connect each platform once, test the credentials, and reuse them for every stream.</p>
                                        </div>
                                        <span>{destinations.length} configured</span>
                                    </div>
                                    {destinationError ? <p className="error-banner">{destinationError}</p> : null}
                                    {destinationStatus ? <p className="success-banner">{destinationStatus}</p> : null}
                                </article>

                                <article className="panel panel-wide">
                                    <div className="panel-header">
                                        <div>
                                            <h2>Saved Destinations</h2>
                                            <p>Pick one to edit, or start a fresh destination setup.</p>
                                        </div>
                                        <div className="form-actions">
                                            <span>{configuredDestinationCount} configured</span>
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
                                                    setDestinationConnectionResult(null);
                                                    setDestinationHelperStatus(null);
                                                }}
                                            >
                                                New Destination
                                            </button>
                                        </div>
                                    </div>

                                    <div className="saved-destinations-grid">
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
                                                        setDestinationConnectionResult(null);
                                                        setDestinationHelperStatus(null);
                                                    }}
                                                >
                                                    <strong>{item.name}</strong>
                                                    <span>{item.platform}</span>
                                                </button>
                                            ))
                                        )}
                                    </div>
                                </article>

                                <article className="panel panel-wide">
                                        <div className="panel-header">
                                            <div>
                                                <h2>{destinationForm.id ? 'Edit Destination' : 'New Destination'}</h2>
                                                <p>Fill in the platform details, test them, then save.</p>
                                            </div>
                                            <div className="form-actions">
                                                <span>Posting setup</span>
                                                <button type="button" className="ghost-button" onClick={() => setShowGuidedSetup(true)}>
                                                    Setup Help
                                                </button>
                                            </div>
                                        </div>

                            <form className="announcement-form" onSubmit={onDestinationSubmit}>
                                <div className="form-grid">
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
                                            setDestinationConnectionResult(null);
                                            setDestinationForm((current) => ({
                                                ...createEmptyDestinationForm(event.target.value as DestinationInput['platform']),
                                                id: current.id,
                                                name: current.name,
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

                                </div>

                                <label className="field">
                                    <span>{destinationTemplateLabel(destinationForm.platform)}</span>
                                    <textarea
                                        aria-label={destinationTemplateLabel(destinationForm.platform)}
                                        value={destinationForm.template}
                                        onChange={(event) => updateDestination('template', event.target.value)}
                                        placeholder="{{stream_title}}"
                                        rows={5}
                                    />
                                </label>

                                <section className="mini-panel template-helper-panel">
                                    <div className="template-helper-header">
                                        <h3>How This Template Works</h3>
                                        <button
                                            type="button"
                                            className="ghost-button"
                                            onClick={() => updateDestination('template', defaultTemplateForPlatform(destinationForm.platform))}
                                        >
                                            Reset to Starter Template
                                        </button>
                                    </div>
                                    <p>Only fields referenced in this template will appear in the published post. The announcement form does not automatically include every field unless the template asks for it.</p>
                                    <p>
                                        Uses:{' '}
                                        {usedTemplateVariables(destinationForm.template).length > 0
                                            ? usedTemplateVariables(destinationForm.template).join(', ')
                                            : 'no announcement fields yet'}
                                    </p>
                                    {destinationHelperStatus ? <p className="success-copy">{destinationHelperStatus}</p> : null}
                                    <div className="template-token-list" role="list" aria-label="Available template variables">
                                        {TEMPLATE_VARIABLES.map((variable) => (
                                            <button
                                                key={variable}
                                                type="button"
                                                className="template-token"
                                                onClick={() => void onCopyTemplateVariable(variable)}
                                            >
                                                {variable}
                                            </button>
                                        ))}
                                    </div>
                                </section>

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
                                            <span>Webhook URL</span>
                                            <input
                                                aria-label="Webhook URL"
                                                value={destinationForm.discordWebhookKey}
                                                onChange={(event) => updateDestination('discordWebhookKey', event.target.value)}
                                                placeholder="https://discord.com/api/webhooks/..."
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
                                            <span>App Password</span>
                                            <input
                                                aria-label="App Password"
                                                value={destinationForm.blueskyCredentialKey}
                                                onChange={(event) => updateDestination('blueskyCredentialKey', event.target.value)}
                                                placeholder="xxxx-xxxx-xxxx-xxxx"
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
                                            <span>Access Token</span>
                                            <input
                                                aria-label="Access Token"
                                                value={destinationForm.mastodonCredentialKey}
                                                onChange={(event) => updateDestination('mastodonCredentialKey', event.target.value)}
                                                placeholder="Paste Mastodon access token"
                                            />
                                        </label>
                                    </>
                                ) : null}

                                <div className="action-cluster">
                                    <div className="action-cluster-main">
                                    <button type="submit" className="primary-button">
                                        Save Destination
                                    </button>
                                    <button
                                        type="button"
                                        className="ghost-button"
                                        onClick={() => void onTestDestinationConnection()}
                                        disabled={destinationConnectionLoading}
                                    >
                                        {destinationConnectionLoading ? 'Testing Connection...' : `Test ${credentialSetupLabel(destinationForm.platform)}`}
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
                                </div>

                                {destinationConnectionResult ? (
                                    destinationConnectionResult.state === 'SUCCESS' ? (
                                        <p className="success-banner">
                                            {destinationConnectionResult.message}
                                        </p>
                                    ) : (
                                        <p className="error-banner">
                                            {destinationConnectionResult.message}
                                        </p>
                                    )
                                ) : null}
                            </form>
                                </article>

                                {showGuidedSetup ? (
                                    <div className="modal-backdrop" role="presentation" onClick={() => setShowGuidedSetup(false)}>
                                        <section
                                            className="panel modal-panel"
                                            role="dialog"
                                            aria-modal="true"
                                            aria-label="Guided Credential Setup"
                                            onClick={(event) => event.stopPropagation()}
                                        >
                                            <div className="panel-header">
                                                <div>
                                                    <h2>{destinationForm.platform.charAt(0).toUpperCase() + destinationForm.platform.slice(1)} Setup</h2>
                                                    <p>Use these steps, then come back and test the destination before saving.</p>
                                                </div>
                                                <button type="button" className="ghost-button" onClick={() => setShowGuidedSetup(false)}>
                                                    Close
                                                </button>
                                            </div>
                                            <div className="guided-setup-list">
                                                {guidedSetupSteps(destinationForm.platform).map((step) => (
                                                    <section className="guided-setup-step" key={step.title}>
                                                        <strong>{step.title}</strong>
                                                        <p>{step.detail}</p>
                                                    </section>
                                                ))}
                                            </div>
                                        </section>
                                    </div>
                                ) : null}
                            </>
                        ) : null}

                        {selectedTab === 'settings' ? (
                            <>
                                <article className="panel panel-wide">
                                    <div className="panel-header">
                                        <div>
                                            <h2>Settings</h2>
                                            <p>Adjust defaults, route safe tests, and manage recovery tools.</p>
                                        </div>
                                        <span>App defaults and safety rails</span>
                                    </div>
                        {settingsError ? <p className="error-banner">{settingsError}</p> : null}
                        {settingsStatus ? <p className="success-banner">{settingsStatus}</p> : null}
                                </article>

                                <section className="two-column-layout">
                                    <article className="panel">
                                        <section className="mini-panel">
                                            <h3>Test Mode Credentials</h3>
                                            <p>These optional values are only used when Test Mode is enabled. They are stored in Windows Credential Manager, separate from your normal posting destinations.</p>
                                        </section>

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
                                <span>Test Discord Webhook URL</span>
                                <input
                                    aria-label="Test Discord Webhook URL"
                                    value={settings.testDiscordWebhookKey}
                                    onChange={(event) => updateSettings('testDiscordWebhookKey', event.target.value)}
                                    placeholder="https://discord.com/api/webhooks/..."
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
                                <span>Test Bluesky App Password</span>
                                <input
                                    aria-label="Test Bluesky App Password"
                                    value={settings.testBlueskyCredentialKey}
                                    onChange={(event) => updateSettings('testBlueskyCredentialKey', event.target.value)}
                                    placeholder="xxxx-xxxx-xxxx-xxxx"
                                />
                            </label>

                            <label className="field">
                                <span>Test Mastodon Access Token</span>
                                <input
                                    aria-label="Test Mastodon Access Token"
                                    value={settings.testMastodonCredentialKey}
                                    onChange={(event) => updateSettings('testMastodonCredentialKey', event.target.value)}
                                    placeholder="Paste Mastodon access token"
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
                                    </article>

                                    <article className="panel">
                                        <section className="mini-panel">
                                            <h3>Live Now Recovery</h3>
                                            <p>Use this if Stream Signal closed before Bluesky Live Now was cleared, or if a pending session needs to be cleaned up manually.</p>
                                            {recoveryError ? <p className="error-banner">{recoveryError}</p> : null}
                                            {recoveryStatus ? <p className="success-banner">{recoveryStatus}</p> : null}

                                            {pendingLiveNowSessions.length === 0 ? (
                                                <div className="empty-state compact-empty-state">
                                                    <h3>No pending Live Now sessions.</h3>
                                                    <p>When recovery is needed, each session will appear here with a one-click clear action.</p>
                                                </div>
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
                                </section>
                            </>
                        ) : null}

                        {selectedTab === 'logs' ? (
                            <article className="panel panel-wide">
                                <div className="panel-header">
                                    <div>
                                        <h2>Logs</h2>
                                        <p>Review recent app activity and export diagnostics when you need support context.</p>
                                    </div>
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
                </section>
            </div>
        </main>
    );
}

export default App;
