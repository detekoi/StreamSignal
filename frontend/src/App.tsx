import { FormEvent, useEffect, useRef, useState } from 'react';
import './App.css';
import {
    clearPendingLiveNowSession,
    deleteDestination,
    endStream,
    errorMessage,
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
const SHOW_DEBUG_WORKFLOW_UI = import.meta.env.DEV || import.meta.env.MODE === 'debug';

const initialAnnouncement: AnnouncementInput = {
    streamTitle: '',
    streamURL: '',
    category: '',
    message: '',
    hashtags: '',
};

const initialDestination = createEmptyDestinationForm();

const defaultSettings: AppSettings = {
    defaultStreamURL: '',
    defaultHashtags: '',
    duplicateProtectionEnabled: true,
    duplicateWindowMinutes: 10,
};

type DestinationSecretCache = Pick<DestinationFormState, 'discordWebhookKey' | 'blueskyCredentialKey' | 'mastodonCredentialKey'>;

type GuidedSetupStep = {
    title: string;
    detail: string;
};

function executionModeLabel(mode: ExecutionSummary['mode']) {
    switch (mode) {
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

function applyAnnouncementDefaults(announcement: AnnouncementInput, settings: AppSettings): AnnouncementInput {
    return {
        ...announcement,
        streamURL: announcement.streamURL.trim() === '' ? settings.defaultStreamURL : announcement.streamURL,
        hashtags: announcement.hashtags.trim() === '' ? settings.defaultHashtags : announcement.hashtags,
    };
}

function asArray<T>(value: T[] | null | undefined): T[] {
    return Array.isArray(value) ? value : [];
}

const BLUESKY_CARD_THUMBNAIL_MAX_BYTES = 1_000_000;
const BLUESKY_ADDITIONAL_IMAGE_MAX_BYTES = 1_000_000;
const DISCORD_CARD_THUMBNAIL_MAX_BYTES = 8_000_000;
const MASTODON_ADDITIONAL_IMAGE_MAX_BYTES = 8_000_000;
const BLUESKY_CARD_THUMBNAIL_MAX_EDGE = 1200;

function fileToDataURL(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result ?? ''));
        reader.onerror = () => reject(reader.error ?? new Error('Unable to read selected image.'));
        reader.readAsDataURL(file);
    });
}

function loadImage(dataURL: string): Promise<HTMLImageElement> {
    return new Promise((resolve, reject) => {
        const image = new Image();
        image.onload = () => resolve(image);
        image.onerror = () => reject(new Error('Unable to load selected image.'));
        image.src = dataURL;
    });
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> {
    return new Promise((resolve, reject) => {
        canvas.toBlob((blob) => {
            if (blob) {
                resolve(blob);
                return;
            }
            reject(new Error('Unable to compress selected image.'));
        }, type, quality);
    });
}

function blobToDataURL(blob: Blob): Promise<string> {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result ?? ''));
        reader.onerror = () => reject(reader.error ?? new Error('Unable to read compressed image.'));
        reader.readAsDataURL(blob);
    });
}

async function prepareCardThumbnail(file: File, maxBytes: number): Promise<{ dataURL: string; compressed: boolean }> {
    const original = await fileToDataURL(file);
    if (file.size <= maxBytes) {
        return { dataURL: original, compressed: false };
    }

    const image = await loadImage(original);
    const scale = Math.min(1, BLUESKY_CARD_THUMBNAIL_MAX_EDGE / Math.max(image.naturalWidth, image.naturalHeight));
    const width = Math.max(1, Math.round(image.naturalWidth * scale));
    const height = Math.max(1, Math.round(image.naturalHeight * scale));
    const canvas = document.createElement('canvas');
    canvas.width = width;
    canvas.height = height;

    const context = canvas.getContext('2d');
    if (!context) {
        throw new Error('Unable to prepare selected image.');
    }
    context.fillStyle = '#ffffff';
    context.fillRect(0, 0, width, height);
    context.drawImage(image, 0, 0, width, height);

    for (const quality of [0.9, 0.82, 0.74, 0.66, 0.58, 0.5]) {
        const blob = await canvasToBlob(canvas, 'image/jpeg', quality);
        if (blob.size <= maxBytes) {
            return { dataURL: await blobToDataURL(blob), compressed: true };
        }
    }

    throw new Error(`Card thumbnail is too large to compress below ${Math.round(maxBytes / 1_000_000)} MB.`);
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
                    title: 'Step 1: Open your Mastodon instance',
                    detail: 'Sign in to the account StreamSignal should post from, then open Preferences > Development. On some instances this is under Settings > Development.',
                },
                {
                    title: 'Step 2: Create a new application',
                    detail: 'Name it StreamSignal. If Mastodon asks for a website, you can leave it blank or use your stream/profile URL.',
                },
                {
                    title: 'Step 3: Set the redirect URI',
                    detail: 'Use urn:ietf:wg:oauth:2.0:oob if the field is required. StreamSignal only needs a manually copied access token and does not run an OAuth redirect server.',
                },
                {
                    title: 'Step 4: Choose scopes',
                    detail: 'Enable write:statuses so StreamSignal can publish posts. If your instance requires account verification for token testing, also enable read:accounts.',
                },
                {
                    title: 'Step 5: Copy the access token',
                    detail: 'After saving the application, copy the generated access token. Paste the full instance URL, like https://mastodon.social, and the access token into StreamSignal.',
                },
                {
                    title: 'Step 6: Test before saving',
                    detail: 'Test Connection checks the token against your Mastodon account so invalid, expired, or under-scoped tokens fail early.',
                },
            ];
        default:
            return [];
    }
}

function destinationTemplateLabel(platform: DestinationInput['platform']) {
    return platform === 'bluesky' ? 'Post Template' : 'Template';
}

function destinationEnvironmentLabel(environment: DestinationFormState['environment']) {
    return environment === 'test' ? 'Test' : 'Production';
}

function destinationEnvironmentFrom(destination: DestinationInput): DestinationFormState['environment'] {
    return toDestinationFormState(destination).environment;
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
    const [executionLoading, setExecutionLoading] = useState<'go_live' | 'end_stream' | null>(null);
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
    const [settingsStatus, setSettingsStatus] = useState<string | null>(null);
    const [settingsError, setSettingsError] = useState<string | null>(null);
    const [logs, setLogs] = useState<LogEntry[]>([]);
    const [logsError, setLogsError] = useState<string | null>(null);
    const [diagnosticsStatus, setDiagnosticsStatus] = useState<string | null>(null);
    const initializedSelection = useRef(false);
    const discordThumbnailInputRef = useRef<HTMLInputElement | null>(null);
    const blueskyThumbnailInputRef = useRef<HTMLInputElement | null>(null);
    const blueskyAdditionalImageInputRef = useRef<HTMLInputElement | null>(null);
    const mastodonAdditionalImageInputRef = useRef<HTMLInputElement | null>(null);

    useEffect(() => {
        void refreshDestinations();
        void refreshSettings();
        if (SHOW_DEBUG_WORKFLOW_UI) {
            void refreshLogs();
        }
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
            setDestinations(asArray(items));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load destinations.');
            setDestinationError(message);
        }
    }

    async function refreshSettings() {
        try {
            const current = await getSettings();
            setSettings(current);
            setAnnouncement((existing) => applyAnnouncementDefaults(existing, current));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load settings.');
            setSettingsError(message);
        }
    }

    async function refreshLogs() {
        try {
            const entries = await getLogs();
            setLogs(asArray(entries));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load logs.');
            setLogsError(message);
        }
    }

    async function refreshPendingLiveNowSessions() {
        try {
            const sessions = await listPendingLiveNowSessions();
            setPendingLiveNowSessions(asArray(sessions));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to load pending Live Now sessions.');
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
            setPreviewItems(asArray(preview));
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to generate previews.');
            setPreviewError(message);
            setPreviewItems([]);
        } finally {
            setPreviewLoading(false);
        }
    }

    async function runGoLive() {
        setExecutionLoading('go_live');
        setExecutionError(null);

        try {
            if (destinations.length > 0 && selectedDestinationIDs.length === 0) {
                setExecutionError('Select at least one destination for this session before running this action.');
                setExecutionSummary(null);
                setPendingGoLiveConfirmation(false);
                return;
            }

            const summary = await goLive(announcement, selectedDestinationIDs);
            if (summary.requiresDuplicateConfirmation) {
                setPendingGoLiveConfirmation(true);
                setExecutionSummary(summary);
                return;
            }
            setPendingGoLiveConfirmation(false);
            setExecutionSummary(summary);
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to run execution.');
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
            const message = errorMessage(err, 'Unable to confirm Go Live.');
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
            if (destinations.length > 0 && selectedDestinationIDs.length === 0) {
                setExecutionError('Select at least one destination for this session before running this action.');
                setExecutionSummary(null);
                return;
            }

            const summary = await endStream(selectedDestinationIDs);
            setExecutionSummary(summary);
            await refreshPendingLiveNowSessions();
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to end stream.');
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
            const message = errorMessage(err, 'Unable to clear pending Live Now session.');
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
            const message = errorMessage(err, 'Unable to save destination.');
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
            const message = errorMessage(err, 'Unable to test destination connection.');
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
            const message = errorMessage(err, 'Unable to delete destination.');
            setDestinationError(message);
        }
    }

    async function onSettingsSubmit(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSettingsError(null);
        setSettingsStatus(null);

        try {
            const saved = await saveSettings(settings);
            setSettings(saved);
            setAnnouncement((existing) => applyAnnouncementDefaults(existing, saved));
            setSettingsStatus('Settings saved.');
        } catch (err: unknown) {
            const message = errorMessage(err, 'Unable to save settings.');
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
            const message = errorMessage(err, 'Unable to copy diagnostics.');
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

    function clearBlueskyThumbnailImage() {
        if (blueskyThumbnailInputRef.current) {
            blueskyThumbnailInputRef.current.value = '';
        }
        updateDestination('blueskyCardThumbnailDataURL', '');
    }

    function clearBlueskyAdditionalImage() {
        if (blueskyAdditionalImageInputRef.current) {
            blueskyAdditionalImageInputRef.current.value = '';
        }
        updateDestination('blueskyAdditionalImageDataURL', '');
    }

    function clearMastodonAdditionalImage() {
        if (mastodonAdditionalImageInputRef.current) {
            mastodonAdditionalImageInputRef.current.value = '';
        }
        updateDestination('mastodonAdditionalImageDataURL', '');
    }

    function clearDiscordThumbnailImage() {
        if (discordThumbnailInputRef.current) {
            discordThumbnailInputRef.current.value = '';
        }
        updateDestination('discordCardThumbnailDataURL', '');
    }

    async function onDiscordThumbnailUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Additional image must be an image.');
            return;
        }

        try {
            const thumbnail = await prepareCardThumbnail(file, DISCORD_CARD_THUMBNAIL_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                discordCardThumbnailURL: '',
                discordCardThumbnailDataURL: thumbnail.dataURL,
            }));
            if (discordThumbnailInputRef.current) {
                discordThumbnailInputRef.current.value = '';
            }
            setDestinationHelperStatus(thumbnail.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (discordThumbnailInputRef.current) {
                discordThumbnailInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    async function onBlueskyThumbnailUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Card thumbnail must be an image.');
            return;
        }

        try {
            const thumbnail = await prepareCardThumbnail(file, BLUESKY_CARD_THUMBNAIL_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                blueskyCardThumbnailURL: '',
                blueskyCardThumbnailDataURL: thumbnail.dataURL,
            }));
            if (blueskyThumbnailInputRef.current) {
                blueskyThumbnailInputRef.current.value = '';
            }
            setDestinationHelperStatus(thumbnail.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (blueskyThumbnailInputRef.current) {
                blueskyThumbnailInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    async function onBlueskyAdditionalImageUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Additional image must be an image.');
            return;
        }

        try {
            const image = await prepareCardThumbnail(file, BLUESKY_ADDITIONAL_IMAGE_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                blueskyAdditionalImageURL: '',
                blueskyAdditionalImageDataURL: image.dataURL,
            }));
            if (blueskyAdditionalImageInputRef.current) {
                blueskyAdditionalImageInputRef.current.value = '';
            }
            setDestinationHelperStatus(image.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (blueskyAdditionalImageInputRef.current) {
                blueskyAdditionalImageInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
    }

    async function onMastodonAdditionalImageUpload(file: File | undefined) {
        setDestinationConnectionResult(null);
        setDestinationHelperStatus(null);
        setDestinationError(null);

        if (!file) {
            return;
        }
        if (!file.type.startsWith('image/')) {
            setDestinationError('Additional image must be an image.');
            return;
        }

        try {
            const image = await prepareCardThumbnail(file, MASTODON_ADDITIONAL_IMAGE_MAX_BYTES);
            setDestinationForm((current) => ({
                ...current,
                mastodonAdditionalImageURL: '',
                mastodonAdditionalImageDataURL: image.dataURL,
            }));
            if (mastodonAdditionalImageInputRef.current) {
                mastodonAdditionalImageInputRef.current.value = '';
            }
            setDestinationHelperStatus(image.compressed ? `Selected and compressed ${file.name}` : `Selected ${file.name}`);
        } catch (err: unknown) {
            if (mastodonAdditionalImageInputRef.current) {
                mastodonAdditionalImageInputRef.current.value = '';
            }
            const message = errorMessage(err, 'Unable to prepare selected image.');
            setDestinationError(message);
        }
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
            const message = errorMessage(err, 'Unable to copy template variable.');
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
                            {SHOW_DEBUG_WORKFLOW_UI ? (
                                <button
                                    aria-label="Logs"
                                    className={selectedTab === 'logs' ? 'tab-button active' : 'tab-button'}
                                    onClick={() => setSelectedTab('logs')}
                                >
                                    Logs
                                </button>
                            ) : null}
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
                                        <div className="panel-header-actions">
                                            <div className="home-status-inline">
                                                {pendingLiveNowSessions.length > 0 ? (
                                                    <span className="status-pill">{pendingLiveNowSessions.length} recovery pending</span>
                                                ) : null}
                                            </div>
                                            <button
                                                type="button"
                                                className="ghost-button small-button"
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

                                    <form className="announcement-form" onSubmit={onPreviewSubmit}>
                                        <div className="form-grid">
                                            <div className="field">
                                                <div className="field-label-row">
                                                    <label htmlFor="stream-title">Stream Title</label>
                                                    <button
                                                        type="button"
                                                        className="info-tooltip"
                                                        aria-label="Stream Title information"
                                                    >
                                                        i
                                                        <span role="tooltip">Required for Bluesky Live Now Use</span>
                                                    </button>
                                                </div>
                                                <input
                                                    id="stream-title"
                                                    aria-label="Stream Title"
                                                    value={announcement.streamTitle}
                                                    onChange={(event) => updateAnnouncement('streamTitle', event.target.value)}
                                                    placeholder="Late Night Variety"
                                                />
                                            </div>

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
                                                                <small>
                                                                    {destination.platform} · {destinationEnvironmentLabel(destinationEnvironmentFrom(destination))} ·{' '}
                                                                    {toDestinationFormState(destination).endStreamEnabled ? 'End Stream on' : 'End Stream off'}
                                                                </small>
                                                            </span>
                                                        </label>
                                                    ))}
                                                </div>
                                            ) : null}
                                        </section>

                                        <div className="workflow-actions">
                                            <button type="submit" className="primary-button" disabled={previewLoading}>
                                                {previewLoading ? 'Generating...' : 'Generate Preview'}
                                            </button>
                                            <button
                                                type="button"
                                                className="primary-button"
                                                onClick={() => void runGoLive()}
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

                                        {!SHOW_DEBUG_WORKFLOW_UI ? (
                                            <section className="release-workflow-status" aria-live="polite">
                                                {executionError ? <p className="error-banner">{executionError}</p> : null}
                                                {executionStatusMessage(executionSummary) ? (
                                                    <div className={executionSummary?.status === 'PARTIAL' ? 'error-banner' : 'warning-banner'}>
                                                        <p>{executionStatusMessage(executionSummary)}</p>
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
                                                {executionSummary && !pendingGoLiveConfirmation && !executionStatusMessage(executionSummary) ? (
                                                    <p className="success-banner">
                                                        {executionSummary.mode === 'end_stream' ? 'End Stream completed.' : 'Go Live completed.'}
                                                    </p>
                                                ) : null}
                                            </section>
                                        ) : null}
                                    </form>
                                </article>

                                <article className="panel">
                                        <div className="panel-header">
                                            <div>
                                                <h2>Preview Panel</h2>
                                                <p>See the exact message each selected destination will receive.</p>
                                            </div>
                                            <span>{previewItems.length} messages</span>
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
                                                <section key={`${item.destinationID}-${item.previewLabel || 'preview'}`} className="preview-card">
                                                    <div className="preview-topline">
                                                        <div>
                                                            <h3>{item.destinationName}</h3>
                                                            <p className="preview-platform">
                                                                {item.previewLabel || 'Preview'} · {item.platform}
                                                            </p>
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

                                {SHOW_DEBUG_WORKFLOW_UI ? (
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
                                    <p>Run Go Live or End Stream to see per-destination execution results.</p>
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
                                ) : null}
                            </>
                        ) : null}

                        {selectedTab === 'destinations' ? (
                            <>
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
                                                Reset Form
                                            </button>
                                        </div>
                                    </div>
                                    {destinationError ? <p className="error-banner">{destinationError}</p> : null}
                                    {destinationStatus ? <p className="success-banner">{destinationStatus}</p> : null}

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
                                                    <span>
                                                        {item.platform} · {destinationEnvironmentLabel(destinationEnvironmentFrom(item))}
                                                    </span>
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
                                                environment: current.environment,
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

                                <label className="field">
                                    <span>Destination Type</span>
                                    <select
                                        aria-label="Destination Type"
                                        value={destinationForm.environment}
                                        onChange={(event) => updateDestination('environment', event.target.value as DestinationFormState['environment'])}
                                    >
                                        <option value="production">Production</option>
                                        <option value="test">Test</option>
                                    </select>
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

                                <section className="mini-panel template-helper-panel">
                                    <div className="template-helper-header">
                                        <h3>End Stream Message</h3>
                                        <label className="checkbox-field">
                                            <input
                                                type="checkbox"
                                                aria-label="End Stream Enabled"
                                                checked={destinationForm.endStreamEnabled}
                                                onChange={(event) => updateDestination('endStreamEnabled', event.target.checked)}
                                            />
                                            <span>Enabled</span>
                                        </label>
                                    </div>
                                    <label className="field">
                                        <span>End Stream Template</span>
                                        <textarea
                                            aria-label="Destination End Stream Template"
                                            value={destinationForm.endStreamTemplate}
                                            onChange={(event) => updateDestination('endStreamTemplate', event.target.value)}
                                            rows={3}
                                        />
                                    </label>
                                    <p>
                                        Uses:{' '}
                                        {usedTemplateVariables(destinationForm.endStreamTemplate).length > 0
                                            ? usedTemplateVariables(destinationForm.endStreamTemplate).join(', ')
                                            : 'no announcement fields yet'}
                                    </p>
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
                                        <label className="field">
                                            <div className="field-label-row">
                                                <span>Additional Image URL</span>
                                                <button
                                                    type="button"
                                                    className="info-tooltip"
                                                    aria-label="Discord additional image information"
                                                >
                                                    i
                                                    <span role="tooltip">Discord may show this image instead of its native link preview.</span>
                                                </button>
                                            </div>
                                            <input
                                                aria-label="Discord Additional Image URL"
                                                value={destinationForm.discordCardThumbnailURL}
                                                onChange={(event) => {
                                                    updateDestination('discordCardThumbnailURL', event.target.value);
                                                    if (event.target.value.trim() !== '') {
                                                        clearDiscordThumbnailImage();
                                                    }
                                                }}
                                                placeholder="https://static-cdn.jtvnw.net/..."
                                            />
                                        </label>
                                        <label className="field">
                                            <div className="field-label-row">
                                                <span>Additional Image Upload</span>
                                                <button
                                                    type="button"
                                                    className="info-tooltip"
                                                    aria-label="Discord additional image upload information"
                                                >
                                                    i
                                                    <span role="tooltip">Discord may show this image instead of its native link preview.</span>
                                                </button>
                                            </div>
                                            <input
                                                ref={discordThumbnailInputRef}
                                                aria-label="Discord Additional Image Upload"
                                                type="file"
                                                accept="image/*"
                                                onChange={(event) => void onDiscordThumbnailUpload(event.target.files?.[0])}
                                            />
                                            <small>Large images are resized before posting.</small>
                                        </label>
                                        {destinationForm.discordCardThumbnailDataURL ? (
                                            <div className="form-actions">
                                                <span>Uploaded image selected</span>
                                                <button
                                                    type="button"
                                                    className="ghost-button"
                                                    onClick={clearDiscordThumbnailImage}
                                                >
                                                    Clear Image
                                                </button>
                                            </div>
                                        ) : null}
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
                                                placeholder="handle.bsky.social"
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
                                        <label className="field">
                                            <span>Live Now Duration (Minutes)</span>
                                            <input
                                                aria-label="Live Now Duration (Minutes)"
                                                type="number"
                                                min="1"
                                                value={destinationForm.blueskyLiveNowDurationMinutes}
                                                onChange={(event) => updateDestination('blueskyLiveNowDurationMinutes', event.target.value)}
                                                placeholder="120"
                                            />
                                        </label>
                                        <label className="field">
                                            <div className="field-label-row">
                                                <span>Preview Card Image URL</span>
                                                <button
                                                    type="button"
                                                    className="info-tooltip"
                                                    aria-label="Bluesky preview card image information"
                                                >
                                                    i
                                                    <span role="tooltip">Bluesky only. Used as the stream link preview card image.</span>
                                                </button>
                                            </div>
                                            <input
                                                aria-label="Bluesky Preview Card Image URL"
                                                value={destinationForm.blueskyCardThumbnailURL}
                                                onChange={(event) => {
                                                    updateDestination('blueskyCardThumbnailURL', event.target.value);
                                                    if (event.target.value.trim() !== '') {
                                                        clearBlueskyThumbnailImage();
                                                    }
                                                }}
                                                placeholder="https://static-cdn.jtvnw.net/..."
                                            />
                                        </label>
                                        <label className="field">
                                            <div className="field-label-row">
                                                <span>Preview Card Image Upload</span>
                                                <button
                                                    type="button"
                                                    className="info-tooltip"
                                                    aria-label="Bluesky preview card image upload information"
                                                >
                                                    i
                                                    <span role="tooltip">Bluesky only. Used as the stream link preview card image.</span>
                                                </button>
                                            </div>
                                            <input
                                                ref={blueskyThumbnailInputRef}
                                                aria-label="Bluesky Preview Card Image Upload"
                                                type="file"
                                                accept="image/*"
                                                onChange={(event) => void onBlueskyThumbnailUpload(event.target.files?.[0])}
                                            />
                                            <small>Large images are resized for Bluesky before posting.</small>
                                        </label>
                                        {destinationForm.blueskyCardThumbnailDataURL ? (
                                            <div className="form-actions">
                                                <span>Uploaded preview card image selected</span>
                                                <button
                                                    type="button"
                                                    className="ghost-button"
                                                    onClick={clearBlueskyThumbnailImage}
                                                >
                                                    Clear Image
                                                </button>
                                            </div>
                                        ) : null}
                                        <label className="field">
                                            <div className="field-label-row">
                                                <span>Additional Image URL</span>
                                                <button
                                                    type="button"
                                                    className="info-tooltip"
                                                    aria-label="Bluesky additional image information"
                                                >
                                                    i
                                                    <span role="tooltip">Attached as a Bluesky image when no stream preview card is posted.</span>
                                                </button>
                                            </div>
                                            <input
                                                aria-label="Bluesky Additional Image URL"
                                                value={destinationForm.blueskyAdditionalImageURL}
                                                onChange={(event) => {
                                                    updateDestination('blueskyAdditionalImageURL', event.target.value);
                                                    if (event.target.value.trim() !== '') {
                                                        clearBlueskyAdditionalImage();
                                                    }
                                                }}
                                                placeholder="https://static-cdn.jtvnw.net/..."
                                            />
                                        </label>
                                        <label className="field">
                                            <div className="field-label-row">
                                                <span>Additional Image Upload</span>
                                                <button
                                                    type="button"
                                                    className="info-tooltip"
                                                    aria-label="Bluesky additional image upload information"
                                                >
                                                    i
                                                    <span role="tooltip">Attached as a Bluesky image when no stream preview card is posted.</span>
                                                </button>
                                            </div>
                                            <input
                                                ref={blueskyAdditionalImageInputRef}
                                                aria-label="Bluesky Additional Image Upload"
                                                type="file"
                                                accept="image/*"
                                                onChange={(event) => void onBlueskyAdditionalImageUpload(event.target.files?.[0])}
                                            />
                                            <small>Large images are resized before posting.</small>
                                        </label>
                                        {destinationForm.blueskyAdditionalImageDataURL ? (
                                            <div className="form-actions">
                                                <span>Uploaded additional image selected</span>
                                                <button
                                                    type="button"
                                                    className="ghost-button"
                                                    onClick={clearBlueskyAdditionalImage}
                                                >
                                                    Clear Image
                                                </button>
                                            </div>
                                        ) : null}
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
                                                placeholder="@streamer@example.social"
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
                                        <label className="field">
                                            <span>Additional Image URL</span>
                                            <input
                                                aria-label="Mastodon Additional Image URL"
                                                value={destinationForm.mastodonAdditionalImageURL}
                                                onChange={(event) => {
                                                    updateDestination('mastodonAdditionalImageURL', event.target.value);
                                                    if (event.target.value.trim() !== '') {
                                                        clearMastodonAdditionalImage();
                                                    }
                                                }}
                                                placeholder="https://static-cdn.jtvnw.net/..."
                                            />
                                        </label>
                                        <label className="field">
                                            <span>Additional Image Upload</span>
                                            <input
                                                ref={mastodonAdditionalImageInputRef}
                                                aria-label="Mastodon Additional Image Upload"
                                                type="file"
                                                accept="image/*"
                                                onChange={(event) => void onMastodonAdditionalImageUpload(event.target.files?.[0])}
                                            />
                                            <small>Large images are resized before posting.</small>
                                        </label>
                                        {destinationForm.mastodonAdditionalImageDataURL ? (
                                            <div className="form-actions">
                                                <span>Uploaded additional image selected</span>
                                                <button
                                                    type="button"
                                                    className="ghost-button"
                                                    onClick={clearMastodonAdditionalImage}
                                                >
                                                    Clear Image
                                                </button>
                                            </div>
                                        ) : null}
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
                                            <h3>App Defaults</h3>
                                            <p>Use test destinations in the Destinations tab when you want to publish to test accounts or channels.</p>
                                        </section>

                                        <form className="announcement-form settings-grid" onSubmit={onSettingsSubmit}>
                            <label className="field checkbox-field">
                                <input
                                    aria-label="Duplicate Protection Enabled"
                                    type="checkbox"
                                    checked={settings.duplicateProtectionEnabled}
                                    onChange={(event) => updateSettings('duplicateProtectionEnabled', event.target.checked)}
                                />
                                <span>Duplicate Protection Enabled</span>
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

                        {SHOW_DEBUG_WORKFLOW_UI && selectedTab === 'logs' ? (
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
