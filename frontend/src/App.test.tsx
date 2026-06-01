/// <reference types="vitest/globals" />
/// <reference types="@testing-library/jest-dom" />
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import App from './App';
import * as api from './lib/api/streamsignal';
import type { PreviewItem } from './types/preview';
import type { AppSettings } from './types/settings';

const STORED_SECRET_TOKEN = '[stored securely]';

vi.mock('./lib/api/streamsignal', () => ({
    clearPendingLiveNowSession: vi.fn(),
    deleteDestination: vi.fn(),
    endStream: vi.fn(),
    errorMessage: (err: unknown, fallback: string) => {
        if (err instanceof Error) {
            return err.message;
        }
        if (typeof err === 'string') {
            return err;
        }
        return fallback;
    },
    forceGoLive: vi.fn(),
    getDiagnostics: vi.fn(),
    getLogs: vi.fn(),
    getSettings: vi.fn(),
    generatePreview: vi.fn(),
    goLive: vi.fn(),
    listPendingLiveNowSessions: vi.fn(),
    listDestinations: vi.fn(),
    saveDestination: vi.fn(),
    saveSettings: vi.fn(),
    testDestinationConnection: vi.fn(),
}));

const mockedClearPendingLiveNowSession = vi.mocked(api.clearPendingLiveNowSession);
const mockedDeleteDestination = vi.mocked(api.deleteDestination);
const mockedEndStream = vi.mocked(api.endStream);
const mockedForceGoLive = vi.mocked(api.forceGoLive);
const mockedGetDiagnostics = vi.mocked(api.getDiagnostics);
const mockedGetLogs = vi.mocked(api.getLogs);
const mockedGetSettings = vi.mocked(api.getSettings);
const mockedGeneratePreview = vi.mocked(api.generatePreview);
const mockedGoLive = vi.mocked(api.goLive);
const mockedListPendingLiveNowSessions = vi.mocked(api.listPendingLiveNowSessions);
const mockedListDestinations = vi.mocked(api.listDestinations);
const mockedSaveDestination = vi.mocked(api.saveDestination);
const mockedSaveSettings = vi.mocked(api.saveSettings);
const mockedTestDestinationConnection = vi.mocked(api.testDestinationConnection);

function emptyExecutionSummary(mode: 'go_live' | 'end_stream') {
    return {
        mode,
        status: 'SUCCESS' as const,
        testModeActive: false,
        results: [],
        totalCount: 0,
        successCount: 0,
        failedCount: 0,
        skippedCount: 0,
        validationErrorCount: 0,
        requiresDuplicateConfirmation: false,
        duplicateWarningMessage: '',
    };
}

function deferred<T>() {
    let resolve!: (value: T | PromiseLike<T>) => void;
    let reject!: (reason?: unknown) => void;

    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });

    return { promise, resolve, reject };
}

describe('App', () => {
    beforeEach(() => {
        mockedGetSettings.mockResolvedValue({
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
        });
        mockedGetLogs.mockResolvedValue([]);
        mockedGetDiagnostics.mockResolvedValue('StreamSignal Diagnostics');
        mockedGeneratePreview.mockResolvedValue([]);
        mockedGoLive.mockResolvedValue(emptyExecutionSummary('go_live'));
        mockedForceGoLive.mockResolvedValue(emptyExecutionSummary('go_live'));
        mockedEndStream.mockResolvedValue(emptyExecutionSummary('end_stream'));
        mockedListPendingLiveNowSessions.mockResolvedValue([]);
        mockedClearPendingLiveNowSession.mockResolvedValue({
            destinationID: 'bluesky-main',
            destinationName: 'Main Bluesky',
            platform: 'bluesky',
            state: 'SUCCESS',
            message: 'Recovered and cleared pending Live Now session.',
            content: '',
        });
        mockedListDestinations.mockResolvedValue([]);
        mockedSaveDestination.mockImplementation(async (destination) => ({
            ...destination,
            id: destination.id || 'generated-id',
        }));
        mockedSaveSettings.mockImplementation(async (settings) => settings);
        mockedTestDestinationConnection.mockResolvedValue({
            platform: 'discord',
            state: 'SUCCESS',
            message: 'Discord webhook verified successfully.',
        });
        mockedDeleteDestination.mockResolvedValue();
    });

    afterEach(() => {
        vi.clearAllMocks();
    });

    it('submits the announcement fields to preview generation', async () => {
        mockedGeneratePreview.mockResolvedValue([
            {
                destinationID: 'discord-main',
                destinationName: 'Main Discord',
                platform: 'discord',
                content: 'Going Live https://example.com/live',
                characterCount: 35,
                validationState: 'VALID',
                validationNotes: [],
            },
        ]);

        render(<App />);

        fireEvent.change(await screen.findByLabelText('Stream Title'), {
            target: { value: 'Going Live' },
        });
        fireEvent.change(screen.getByLabelText('Stream URL'), {
            target: { value: 'https://example.com/live' },
        });

        fireEvent.click(screen.getByRole('button', { name: 'Generate Preview' }));

        await waitFor(() => {
            expect(mockedGeneratePreview).toHaveBeenCalledWith(
                {
                    streamTitle: 'Going Live',
                    streamURL: 'https://example.com/live',
                    category: '',
                    message: '',
                    hashtags: '',
                },
                [],
            );
        });

        expect(await screen.findByText('Main Discord')).toBeInTheDocument();
        expect(screen.getByText('VALID')).toBeInTheDocument();
    });

    it('shows preview validation notes from the backend', async () => {
        mockedGeneratePreview.mockResolvedValue([
            {
                destinationID: 'bluesky-main',
                destinationName: 'Main Bluesky',
                platform: 'bluesky',
                content: '',
                characterCount: 0,
                validationState: 'INVALID',
                validationNotes: ['Stream title is required.', 'Rendered content is empty.'],
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Generate Preview' }));

        expect(await screen.findByText('INVALID')).toBeInTheDocument();
        expect(screen.getByText('Stream title is required.')).toBeInTheDocument();
        expect(screen.getByText('Rendered content is empty.')).toBeInTheDocument();
    });

    it('shows a loading state while preview generation is pending', async () => {
        const pendingPreview = deferred<PreviewItem[]>();
        mockedGeneratePreview.mockReturnValue(pendingPreview.promise);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Generate Preview' }));

        expect(screen.getByRole('button', { name: 'Generating...' })).toBeDisabled();

        pendingPreview.resolve([
            {
                destinationID: 'discord-main',
                destinationName: 'Main Discord',
                platform: 'discord',
                content: 'Ready',
                characterCount: 5,
                validationState: 'VALID',
                validationNotes: [],
            },
        ]);

        expect(await screen.findByText('Main Discord')).toBeInTheDocument();
        await waitFor(() => {
            expect(screen.getByRole('button', { name: 'Generate Preview' })).not.toBeDisabled();
        });
    });

    it('resets the form and clears the preview panel state', async () => {
        mockedGeneratePreview.mockResolvedValue([
            {
                destinationID: 'discord-main',
                destinationName: 'Main Discord',
                platform: 'discord',
                content: 'Going Live',
                characterCount: 10,
                validationState: 'VALID',
                validationNotes: [],
            },
        ]);

        render(<App />);

        fireEvent.change(await screen.findByLabelText('Stream Title'), {
            target: { value: 'Going Live' },
        });

        fireEvent.click(screen.getByRole('button', { name: 'Generate Preview' }));
        expect(await screen.findByText('Main Discord')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Reset' }));

        expect(screen.getByLabelText('Stream Title')).toHaveValue('');
        expect(screen.queryByText('Main Discord')).not.toBeInTheDocument();
        expect(screen.getByText('No previews yet')).toBeInTheDocument();
    });

    it('shows a preview error when the backend request fails', async () => {
        mockedGeneratePreview.mockRejectedValue(new Error('Preview service unavailable.'));

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Generate Preview' }));

        expect(await screen.findByText('Preview service unavailable.')).toBeInTheDocument();
        expect(screen.getByText('No previews yet')).toBeInTheDocument();
    });

    it('shows the compact app header on the Home tab', async () => {
        render(<App />);

        expect(await screen.findByText('StreamSignal')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'Generate Preview' })).toBeInTheDocument();
        expect(screen.getByText('No previews yet')).toBeInTheDocument();
        expect(screen.getByText('Add at least one destination first, then generate a preview to see per-destination content.')).toBeInTheDocument();
    });

    it('loads saved announcement defaults into empty Home fields', async () => {
        mockedGetSettings.mockResolvedValue({
            testModeEnabled: false,
            testDiscordWebhookKey: '',
            testBlueskyAccountIdentifier: '',
            testBlueskyCredentialKey: '',
            testMastodonCredentialKey: '',
            testMastodonInstanceURL: '',
            defaultStreamURL: 'https://example.com/live',
            defaultHashtags: '#vtuber',
            duplicateProtectionEnabled: true,
            duplicateWindowMinutes: 10,
            endStreamPostEnabled: false,
            endStreamTemplate: '',
        });

        render(<App />);

        expect(await screen.findByLabelText('Stream URL')).toHaveValue('https://example.com/live');
        expect(screen.getByLabelText('Hashtags')).toHaveValue('#vtuber');
    });

    it('does not overwrite Home fields typed before settings finish loading', async () => {
        const pendingSettings = deferred<AppSettings>();
        mockedGetSettings.mockReturnValue(pendingSettings.promise);

        render(<App />);

        fireEvent.change(await screen.findByLabelText('Stream URL'), {
            target: { value: 'https://typed.example/live' },
        });

        pendingSettings.resolve({
            testModeEnabled: false,
            testDiscordWebhookKey: '',
            testBlueskyAccountIdentifier: '',
            testBlueskyCredentialKey: '',
            testMastodonCredentialKey: '',
            testMastodonInstanceURL: '',
            defaultStreamURL: 'https://default.example/live',
            defaultHashtags: '#default',
            duplicateProtectionEnabled: true,
            duplicateWindowMinutes: 10,
            endStreamPostEnabled: false,
            endStreamTemplate: '',
        });

        await waitFor(() => {
            expect(screen.getByLabelText('Stream URL')).toHaveValue('https://typed.example/live');
        });
        expect(screen.getByLabelText('Hashtags')).toHaveValue('#default');
    });

    it('loads destination items into the Destinations tab', async () => {
        mockedListDestinations.mockResolvedValue([
            {
                id: 'discord-main',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"webhookKey":"discord/main"}',
                createdAt: '',
                updatedAt: '',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));

        expect(await screen.findByText('Main Discord')).toBeInTheDocument();
        expect(screen.getByText('discord · Production')).toBeInTheDocument();
        expect(screen.getByRole('button', { name: 'New Destination' })).toBeInTheDocument();
    });

    it('treats a null destination list from Wails as empty', async () => {
        mockedListDestinations.mockResolvedValue(null as never);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));

        expect(await screen.findByText('No destinations yet')).toBeInTheDocument();
        expect(screen.getAllByText('0 configured').length).toBeGreaterThan(0);
    });

    it('uses the Home tab destination selection for preview generation', async () => {
        mockedListDestinations.mockResolvedValue([
            {
                id: 'discord-main',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"webhookKey":"discord/main"}',
                createdAt: '',
                updatedAt: '',
            },
            {
                id: 'bluesky-main',
                platform: 'bluesky',
                name: 'Main Bluesky',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"accountIdentifier":"don.test","credentialKey":"bluesky/main"}',
                createdAt: '',
                updatedAt: '',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByLabelText('Main Bluesky'));
        fireEvent.click(screen.getByRole('button', { name: 'Generate Preview' }));

        await waitFor(() => {
            expect(mockedGeneratePreview).toHaveBeenCalledWith(
                {
                    streamTitle: '',
                    streamURL: '',
                    category: '',
                    message: '',
                    hashtags: '',
                },
                ['discord-main'],
            );
        });
    });

    it('uses the Home tab destination selection for End Stream', async () => {
        mockedListDestinations.mockResolvedValue([
            {
                id: 'discord-main',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"webhookKey":"discord/main"}',
                createdAt: '',
                updatedAt: '',
            },
            {
                id: 'bluesky-main',
                platform: 'bluesky',
                name: 'Main Bluesky',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"accountIdentifier":"don.test","credentialKey":"bluesky/main"}',
                createdAt: '',
                updatedAt: '',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByLabelText('Main Bluesky'));
        fireEvent.click(screen.getByRole('button', { name: 'End Stream' }));

        await waitFor(() => {
            expect(mockedEndStream).toHaveBeenCalledWith(['discord-main']);
        });
    });

    it('hydrates platform-specific destination fields from config json', async () => {
        mockedListDestinations.mockResolvedValue([
            {
                id: 'discord-main',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"serverName":"My Server","channelName":"go-live","webhookKey":"discord/main"}',
                createdAt: '',
                updatedAt: '',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));
        fireEvent.click(await screen.findByRole('button', { name: /Main Discord/i }));

        expect(await screen.findByLabelText('Server Name')).toHaveValue('My Server');
        expect(screen.getByLabelText('Channel Name')).toHaveValue('go-live');
        expect(screen.getByLabelText('Webhook URL')).toHaveValue(STORED_SECRET_TOKEN);
    });

    it('saves a destination from the Destinations tab form', async () => {
        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));

        fireEvent.change(await screen.findByLabelText('Friendly Name'), {
            target: { value: 'Main Discord' },
        });
        fireEvent.change(screen.getByLabelText('Template'), {
            target: { value: '{{stream_title}}' },
        });
        fireEvent.change(screen.getByLabelText('Server Name'), {
            target: { value: 'My Server' },
        });
        fireEvent.change(screen.getByLabelText('Channel Name'), {
            target: { value: 'go-live' },
        });
        fireEvent.change(screen.getByLabelText('Webhook URL'), {
            target: { value: 'discord/main' },
        });

        fireEvent.click(screen.getByRole('button', { name: 'Save Destination' }));

        await waitFor(() => {
            expect(mockedSaveDestination).toHaveBeenCalledWith({
                id: '',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"environment":"production","serverName":"My Server","channelName":"go-live","webhookKey":"discord/main","cardThumbnailURL":"","cardThumbnailDataURL":"","endStreamEnabled":false,"endStreamTemplate":"Thanks for hanging out at {{stream_title}} {{stream_url}}"}',
                createdAt: '',
                updatedAt: '',
            });
        });

        expect(await screen.findByText('Destination saved.')).toBeInTheDocument();
    });

    it('tests a discord destination connection before saving', async () => {
        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));

        expect(screen.getByText(/Only fields referenced in this template will appear/i)).toBeInTheDocument();

        fireEvent.change(screen.getByLabelText('Friendly Name'), {
            target: { value: 'Main Discord' },
        });
        fireEvent.change(screen.getByLabelText('Template'), {
            target: { value: '{{stream_title}}' },
        });
        fireEvent.change(screen.getByLabelText('Webhook URL'), {
            target: { value: 'https://discord.com/api/webhooks/test' },
        });

        fireEvent.click(screen.getByRole('button', { name: 'Test Discord webhook' }));

        await waitFor(() => {
            expect(mockedTestDestinationConnection).toHaveBeenCalledWith({
                id: '',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"environment":"production","serverName":"","channelName":"","webhookKey":"https://discord.com/api/webhooks/test","cardThumbnailURL":"","cardThumbnailDataURL":"","endStreamEnabled":false,"endStreamTemplate":"Thanks for hanging out at {{stream_title}} {{stream_url}}"}',
                createdAt: '',
                updatedAt: '',
            });
        });

        expect(await screen.findByText('Discord webhook verified successfully.')).toBeInTheDocument();
    });

    it('shows which announcement fields a destination template uses and can reset to the starter template', async () => {
        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));

        expect(await screen.findByText('Uses: title, stream URL, message, hashtags')).toBeInTheDocument();

        fireEvent.change(screen.getByLabelText('Template'), {
            target: { value: '{{stream_title}}' },
        });

        expect(await screen.findByText('Uses: title')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Reset to Starter Template' }));

        expect(screen.getByLabelText('Template')).toHaveValue('{{stream_title}}\n{{stream_url}}\n{{message}}\n{{hashtags}}');
        expect(screen.getByText('Uses: title, stream URL, message, hashtags')).toBeInTheDocument();
    });

    it('opens guided setup in a modal instead of leaving it expanded on the page', async () => {
        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));
        fireEvent.click(screen.getByRole('button', { name: 'Setup Help' }));

        expect(await screen.findByRole('dialog', { name: 'Guided Credential Setup' })).toBeInTheDocument();
        expect(screen.getByText('Step 1: Create a channel webhook')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Close' }));

        await waitFor(() => {
            expect(screen.queryByRole('dialog', { name: 'Guided Credential Setup' })).not.toBeInTheDocument();
        });
    });

    it('copies template variables from the destination helper', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined);
        Object.assign(navigator, {
            clipboard: {
                writeText,
            },
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));
        fireEvent.click(screen.getByRole('button', { name: '{{stream_url}}' }));

        await waitFor(() => {
            expect(writeText).toHaveBeenCalledWith('{{stream_url}}');
        });

        expect(await screen.findByText('Copied {{stream_url}}')).toBeInTheDocument();
    });

    it('shows validation feedback when destination connection testing fails', async () => {
        mockedTestDestinationConnection.mockResolvedValue({
            platform: 'bluesky',
            state: 'VALIDATION_ERROR',
            message: 'Bluesky account identifier is required.',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));
        fireEvent.change(await screen.findByLabelText('Platform'), {
            target: { value: 'bluesky' },
        });

        fireEvent.click(screen.getByRole('button', { name: 'Setup Help' }));
        expect(await screen.findByText('Step 1: Create an app password')).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', { name: 'Close' }));

        fireEvent.click(screen.getByRole('button', { name: 'Test Bluesky app password' }));

        expect(await screen.findByText('Bluesky account identifier is required.')).toBeInTheDocument();
    });

    it('loads and saves settings from the Settings tab', async () => {
        mockedGetSettings.mockResolvedValue({
            testModeEnabled: false,
            testDiscordWebhookKey: '',
            testBlueskyAccountIdentifier: '',
            testBlueskyCredentialKey: '',
            testMastodonCredentialKey: '',
            testMastodonInstanceURL: '',
            defaultStreamURL: 'https://example.com/live',
            defaultHashtags: '#vtuber',
            duplicateProtectionEnabled: true,
            duplicateWindowMinutes: 10,
            endStreamPostEnabled: false,
            endStreamTemplate: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));

        expect(await screen.findByLabelText('Default Stream URL')).toHaveValue('https://example.com/live');

        fireEvent.change(screen.getByLabelText('Duplicate Window (Minutes)'), {
            target: { value: '15' },
        });
        fireEvent.click(screen.getByRole('button', { name: 'Save Settings' }));

        await waitFor(() => {
            expect(mockedSaveSettings).toHaveBeenCalledWith({
                testModeEnabled: false,
                testDiscordWebhookKey: '',
                testBlueskyAccountIdentifier: '',
                testBlueskyCredentialKey: '',
                testMastodonCredentialKey: '',
                testMastodonInstanceURL: '',
                defaultStreamURL: 'https://example.com/live',
                defaultHashtags: '#vtuber',
                duplicateProtectionEnabled: true,
                duplicateWindowMinutes: 15,
                endStreamPostEnabled: false,
                endStreamTemplate: '',
            });
        });

        expect(await screen.findByText('Settings saved.')).toBeInTheDocument();
    });

    it('preserves hidden legacy test settings when settings are saved', async () => {
        mockedGetSettings.mockResolvedValue({
            testModeEnabled: true,
            testDiscordWebhookKey: 'discord/test',
            testBlueskyAccountIdentifier: 'don.test',
            testBlueskyCredentialKey: 'bluesky/test',
            testMastodonCredentialKey: 'mastodon/test',
            testMastodonInstanceURL: 'https://mastodon.test',
            defaultStreamURL: 'https://example.com/live',
            defaultHashtags: '#vtuber',
            duplicateProtectionEnabled: true,
            duplicateWindowMinutes: 10,
            endStreamPostEnabled: false,
            endStreamTemplate: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));
        fireEvent.click(await screen.findByRole('button', { name: 'Save Settings' }));

        await waitFor(() => {
            expect(mockedSaveSettings).toHaveBeenCalledWith({
                testModeEnabled: true,
                testDiscordWebhookKey: 'discord/test',
                testBlueskyAccountIdentifier: 'don.test',
                testBlueskyCredentialKey: 'bluesky/test',
                testMastodonCredentialKey: 'mastodon/test',
                testMastodonInstanceURL: 'https://mastodon.test',
                defaultStreamURL: 'https://example.com/live',
                defaultHashtags: '#vtuber',
                duplicateProtectionEnabled: true,
                duplicateWindowMinutes: 10,
                endStreamPostEnabled: false,
                endStreamTemplate: '',
            });
        });
    });

    it('preserves stored destination secrets when saving a masked destination', async () => {
        mockedListDestinations.mockResolvedValue([
            {
                id: 'discord-main',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"serverName":"My Server","channelName":"go-live","webhookKey":"discord/main"}',
                createdAt: '',
                updatedAt: '',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Destinations' }));
        fireEvent.click(await screen.findByRole('button', { name: /Main Discord/i }));
        fireEvent.click(await screen.findByRole('button', { name: 'Save Destination' }));

        await waitFor(() => {
            expect(mockedSaveDestination).toHaveBeenCalledWith({
                id: 'discord-main',
                platform: 'discord',
                name: 'Main Discord',
                enabled: true,
                template: '{{stream_title}}',
                configJSON: '{"environment":"production","serverName":"My Server","channelName":"go-live","webhookKey":"discord/main","cardThumbnailURL":"","cardThumbnailDataURL":"","endStreamEnabled":false,"endStreamTemplate":"Thanks for hanging out at {{stream_title}} {{stream_url}}"}',
                createdAt: '',
                updatedAt: '',
            });
        });
    });

    it('shows and clears pending Live Now recovery sessions from Settings', async () => {
        mockedListPendingLiveNowSessions.mockResolvedValue([
            {
                destinationID: 'bluesky-main',
                destinationName: 'Main Bluesky',
                platform: 'bluesky',
                accountIdentifier: 'don.main',
                credentialKey: 'bluesky/main',
                streamURL: 'https://example.com/live',
                streamTitle: 'Going Live',
                startedAt: '2026-05-31T18:00:00Z',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));

        expect(await screen.findByText('Live Now Recovery')).toBeInTheDocument();
        expect(screen.getByText('Going Live')).toBeInTheDocument();

        fireEvent.click(screen.getByRole('button', { name: 'Recover and Clear Live Now' }));

        await waitFor(() => {
            expect(mockedClearPendingLiveNowSession).toHaveBeenCalledWith('bluesky-main');
        });

        expect(await screen.findByText('Recovered and cleared pending Live Now session.')).toBeInTheDocument();
    });

    it('shows a recovery error when clearing a pending Live Now session fails', async () => {
        mockedListPendingLiveNowSessions.mockResolvedValue([
            {
                destinationID: 'bluesky-main',
                destinationName: 'Main Bluesky',
                platform: 'bluesky',
                accountIdentifier: 'don.main',
                credentialKey: 'bluesky/main',
                streamURL: 'https://example.com/live',
                streamTitle: 'Going Live',
                startedAt: '2026-05-31T18:00:00Z',
            },
        ]);
        mockedClearPendingLiveNowSession.mockRejectedValue(new Error('Recovery clear failed.'));

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));
        fireEvent.click(await screen.findByRole('button', { name: 'Recover and Clear Live Now' }));

        expect(await screen.findByText('Recovery clear failed.')).toBeInTheDocument();
    });

    it('shows a recovery error when the backend returns a failed recovery result', async () => {
        mockedListPendingLiveNowSessions.mockResolvedValue([
            {
                destinationID: 'bluesky-main',
                destinationName: 'Main Bluesky',
                platform: 'bluesky',
                accountIdentifier: 'don.main',
                credentialKey: 'bluesky/main',
                streamURL: 'https://example.com/live',
                streamTitle: 'Going Live',
                startedAt: '2026-05-31T18:00:00Z',
            },
        ]);
        mockedClearPendingLiveNowSession.mockResolvedValue({
            destinationID: 'bluesky-main',
            destinationName: 'Main Bluesky',
            platform: 'bluesky',
            state: 'FAILED',
            message: 'Live Now recovery clear failed: clear denied',
            content: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Settings' }));
        fireEvent.click(await screen.findByRole('button', { name: 'Recover and Clear Live Now' }));

        expect(await screen.findByText('Live Now recovery clear failed: clear denied')).toBeInTheDocument();
        expect(screen.queryByText('Recovered and cleared pending Live Now session.')).not.toBeInTheDocument();
    });

    it('loads logs into the Logs tab', async () => {
        mockedGetLogs.mockResolvedValue([
            {
                timestamp: '2026-05-31T12:00:00Z',
                destination: 'preview',
                action: 'generate_preview',
                status: 'SUCCESS',
                message: 'Generated 1 preview items.',
            },
        ]);

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Logs' }));

        expect(await screen.findByText('Generated 1 preview items.')).toBeInTheDocument();
        expect(screen.getByText('Preview')).toBeInTheDocument();
    });

    it('copies diagnostics from the Logs tab', async () => {
        const writeText = vi.fn().mockResolvedValue(undefined);
        Object.assign(navigator, {
            clipboard: {
                writeText,
            },
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Logs' }));
        fireEvent.click(screen.getByRole('button', { name: 'Copy Diagnostics' }));

        await waitFor(() => {
            expect(mockedGetDiagnostics).toHaveBeenCalled();
            expect(writeText).toHaveBeenCalledWith('StreamSignal Diagnostics');
        });

        expect(await screen.findByText('Diagnostics copied.')).toBeInTheDocument();
    });

    it('shows an error when clipboard support is unavailable for diagnostics copy', async () => {
        Object.assign(navigator, {
            clipboard: undefined,
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Logs' }));
        fireEvent.click(screen.getByRole('button', { name: 'Copy Diagnostics' }));

        expect(await screen.findByText('Clipboard is unavailable in this environment.')).toBeInTheDocument();
    });

    it('runs Go Live from the Home tab and shows per-destination failures', async () => {
        mockedGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'WARNING',
            testModeActive: false,
            results: [
                {
                    destinationID: 'discord-main',
                    destinationName: 'Main Discord',
                    platform: 'discord',
                    state: 'SKIPPED',
                    message: 'Discord integration is not connected yet',
                    content: 'Going Live',
                },
            ],
            totalCount: 1,
            successCount: 0,
            failedCount: 0,
            skippedCount: 1,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Go Live' }));

        await waitFor(() => {
            expect(mockedGoLive).toHaveBeenCalled();
        });

        expect(await screen.findByText('Discord integration is not connected yet')).toBeInTheDocument();
        expect(screen.getByText('SKIPPED')).toBeInTheDocument();
        expect(screen.getByText(/completed with warnings\. One or more destinations were skipped/i)).toBeInTheDocument();
    });

    it('shows partial execution guidance when some destinations fail and others need validation', async () => {
        mockedGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'PARTIAL',
            testModeActive: false,
            results: [
                {
                    destinationID: 'discord-main',
                    destinationName: 'Main Discord',
                    platform: 'discord',
                    state: 'FAILED',
                    message: 'Discord request failed.',
                    content: 'Going Live',
                },
                {
                    destinationID: 'bluesky-main',
                    destinationName: 'Main Bluesky',
                    platform: 'bluesky',
                    state: 'VALIDATION_ERROR',
                    message: 'Stream URL is required.',
                    content: '',
                },
            ],
            totalCount: 2,
            successCount: 0,
            failedCount: 1,
            skippedCount: 0,
            validationErrorCount: 1,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Go Live' }));

        expect(
            await screen.findByText(
                'Execution finished with issues. Some destinations failed after execution began, and others still need validation before retrying.',
            ),
        ).toBeInTheDocument();
        expect(screen.getByText('Discord request failed.')).toBeInTheDocument();
        expect(screen.getByText('Stream URL is required.')).toBeInTheDocument();
    });

    it('shows partial execution guidance when destinations fail after execution begins', async () => {
        mockedGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'PARTIAL',
            testModeActive: false,
            results: [
                {
                    destinationID: 'discord-main',
                    destinationName: 'Main Discord',
                    platform: 'discord',
                    state: 'FAILED',
                    message: 'Discord request failed.',
                    content: 'Going Live',
                },
            ],
            totalCount: 1,
            successCount: 0,
            failedCount: 1,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Go Live' }));

        expect(
            await screen.findByText('Execution finished with issues. Some destinations failed after execution began.'),
        ).toBeInTheDocument();
    });

    it('shows partial execution guidance when destinations still need validation', async () => {
        mockedGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'PARTIAL',
            testModeActive: false,
            results: [
                {
                    destinationID: 'bluesky-main',
                    destinationName: 'Main Bluesky',
                    platform: 'bluesky',
                    state: 'VALIDATION_ERROR',
                    message: 'Stream URL is required.',
                    content: '',
                },
            ],
            totalCount: 1,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 1,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Go Live' }));

        expect(
            await screen.findByText('Execution needs attention. Some destinations still have validation issues before they can run.'),
        ).toBeInTheDocument();
    });

    it('labels end-stream post rows distinctly from Live Now clear rows', async () => {
        mockedEndStream.mockResolvedValue({
            mode: 'end_stream',
            status: 'SUCCESS',
            testModeActive: false,
            results: [
                {
                    destinationID: 'mastodon-main',
                    destinationName: 'Main Mastodon',
                    platform: 'mastodon',
                    state: 'SUCCESS',
                    message: 'End stream post published successfully.',
                    content: 'Thanks for hanging out!',
                },
            ],
            totalCount: 1,
            successCount: 1,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'End Stream' }));

        expect(await screen.findByText('End stream post published successfully.')).toBeInTheDocument();
        expect(screen.getByText('End Stream Post · mastodon')).toBeInTheDocument();
    });

    it('shows Bluesky Live Now messaging after Go Live succeeds', async () => {
        mockedGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'SUCCESS',
            testModeActive: false,
            results: [
                {
                    destinationID: 'bluesky-main',
                    destinationName: 'Main Bluesky',
                    platform: 'bluesky',
                    state: 'SUCCESS',
                    message: 'Published successfully. Live Now set.',
                    content: 'Going Live',
                },
            ],
            totalCount: 1,
            successCount: 1,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Go Live' }));

        expect(await screen.findByText('Published successfully. Live Now set.')).toBeInTheDocument();
        expect(screen.getByText('Go Live + Live Now · bluesky')).toBeInTheDocument();
    });

    it('runs End Stream and shows Live Now clear results', async () => {
        mockedEndStream.mockResolvedValue({
            mode: 'end_stream',
            status: 'SUCCESS',
            testModeActive: false,
            results: [
                {
                    destinationID: 'bluesky-main',
                    destinationName: 'Main Bluesky',
                    platform: 'bluesky',
                    state: 'SUCCESS',
                    message: 'Live Now cleared successfully.',
                    content: '',
                },
            ],
            totalCount: 1,
            successCount: 1,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'End Stream' }));

        await waitFor(() => {
            expect(mockedEndStream).toHaveBeenCalled();
        });

        expect(await screen.findByText('Live Now cleared successfully.')).toBeInTheDocument();
        expect(screen.getByText('Live Now Clear · bluesky')).toBeInTheDocument();
    });

    it('shows duplicate confirmation before confirmed Go Live execution', async () => {
        mockedGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'PARTIAL',
            testModeActive: false,
            results: [
                {
                    destinationID: 'discord-invalid',
                    destinationName: 'Invalid Discord',
                    platform: 'discord',
                    state: 'VALIDATION_ERROR',
                    message: 'Rendered content is empty.',
                    content: '',
                },
            ],
            totalCount: 1,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 1,
            requiresDuplicateConfirmation: true,
            duplicateWarningMessage: 'A similar announcement was recently posted within the last 10 minutes. Continue?',
        });
        mockedForceGoLive.mockResolvedValue({
            mode: 'go_live',
            status: 'WARNING',
            testModeActive: false,
            results: [
                {
                    destinationID: 'discord-main',
                    destinationName: 'Main Discord',
                    platform: 'discord',
                    state: 'SKIPPED',
                    message: 'Discord integration is not connected yet',
                    content: 'Going Live',
                },
            ],
            totalCount: 1,
            successCount: 0,
            failedCount: 0,
            skippedCount: 1,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        });

        render(<App />);

        fireEvent.click(await screen.findByRole('button', { name: 'Go Live' }));

        expect(await screen.findByText(/A similar announcement was recently posted/i)).toBeInTheDocument();
        expect(screen.getByText('Rendered content is empty.')).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', { name: 'Confirm Go Live' }));

        await waitFor(() => {
            expect(mockedForceGoLive).toHaveBeenCalledWith(
                {
                    streamTitle: '',
                    streamURL: '',
                    category: '',
                    message: '',
                    hashtags: '',
                },
                [],
            );
        });

        expect(await screen.findByText('Discord integration is not connected yet')).toBeInTheDocument();
    });
});
