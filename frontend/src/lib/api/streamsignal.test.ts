import { describe, expect, it, vi, beforeEach } from 'vitest';

vi.mock('../../../wailsjs/go/main/App', () => ({
    ClearPendingLiveNowSession: vi.fn(),
    DeleteDestination: vi.fn(),
    DryRun: vi.fn(),
    EndStream: vi.fn(),
    ForceGoLive: vi.fn(),
    GeneratePreview: vi.fn(),
    GetDiagnostics: vi.fn(),
    GetLogs: vi.fn(),
    GetSettings: vi.fn(),
    GoLive: vi.fn(),
    ListPendingLiveNowSessions: vi.fn(),
    ListDestinations: vi.fn(),
    SaveDestination: vi.fn(),
    SaveSettings: vi.fn(),
    TestDestinationConnection: vi.fn(),
}));

import * as bindings from '../../../wailsjs/go/main/App';
import {
    clearPendingLiveNowSession,
    deleteDestination,
    dryRun,
    endStream,
    forceGoLive,
    generatePreview,
    getDiagnostics,
    getLogs,
    getSettings,
    goLive,
    listDestinations,
    listPendingLiveNowSessions,
    saveDestination,
    saveSettings,
    testDestinationConnection,
} from './streamsignal';

describe('streamsignal api wrappers', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('delegates each wrapper to the matching Wails binding', async () => {
        const announcement = {
            streamTitle: 'Going Live',
            streamURL: 'https://example.com/live',
            category: 'Music',
            message: 'See you there',
            hashtags: '#vtuber',
        };
        const destinationIDs = ['discord-main'];
        const destination = {
            id: 'discord-main',
            platform: 'discord',
            name: 'Main Discord',
            enabled: true,
            template: '{{stream_title}}',
            configJSON: '{"webhookKey":"discord/main"}',
            createdAt: '',
            updatedAt: '',
        };
        const settings = {
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

        vi.mocked(bindings.GetLogs).mockResolvedValue([] as never);
        vi.mocked(bindings.GetDiagnostics).mockResolvedValue('diag' as never);
        vi.mocked(bindings.GeneratePreview).mockResolvedValue([] as never);
        vi.mocked(bindings.DryRun).mockResolvedValue({
            mode: 'dry_run',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.GoLive).mockResolvedValue({
            mode: 'go_live',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.ForceGoLive).mockResolvedValue({
            mode: 'go_live',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.EndStream).mockResolvedValue({
            mode: 'end_stream',
            status: 'SUCCESS',
            testModeActive: false,
            results: [],
            totalCount: 0,
            successCount: 0,
            failedCount: 0,
            skippedCount: 0,
            validationErrorCount: 0,
            requiresDuplicateConfirmation: false,
            duplicateWarningMessage: '',
        } as never);
        vi.mocked(bindings.ListPendingLiveNowSessions).mockResolvedValue([] as never);
        vi.mocked(bindings.ClearPendingLiveNowSession).mockResolvedValue({
            destinationID: 'bluesky-main',
            destinationName: 'Main Bluesky',
            platform: 'bluesky',
            state: 'SUCCESS',
            message: 'Recovered and cleared pending Live Now session.',
            content: '',
        } as never);
        vi.mocked(bindings.ListDestinations).mockResolvedValue([] as never);
        vi.mocked(bindings.SaveDestination).mockResolvedValue(destination as never);
        vi.mocked(bindings.DeleteDestination).mockResolvedValue(undefined as never);
        vi.mocked(bindings.GetSettings).mockResolvedValue(settings as never);
        vi.mocked(bindings.SaveSettings).mockResolvedValue(settings as never);
        vi.mocked(bindings.TestDestinationConnection).mockResolvedValue({
            platform: 'discord',
            state: 'SUCCESS',
            message: 'Connection verified.',
        } as never);

        await getLogs();
        await getDiagnostics();
        await generatePreview(announcement, destinationIDs);
        await dryRun(announcement, destinationIDs);
        await goLive(announcement, destinationIDs);
        await forceGoLive(announcement, destinationIDs);
        await endStream();
        await listPendingLiveNowSessions();
        await clearPendingLiveNowSession('bluesky-main');
        await listDestinations();
        await saveDestination(destination);
        await deleteDestination('discord-main');
        await getSettings();
        await saveSettings(settings);
        await testDestinationConnection(destination);

        expect(bindings.GetLogs).toHaveBeenCalled();
        expect(bindings.GetDiagnostics).toHaveBeenCalled();
        expect(bindings.GeneratePreview).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationIDs }));
        expect(bindings.DryRun).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationIDs }));
        expect(bindings.GoLive).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationIDs }));
        expect(bindings.ForceGoLive).toHaveBeenCalledWith(expect.objectContaining({ ...announcement, destinationIDs }));
        expect(bindings.EndStream).toHaveBeenCalled();
        expect(bindings.ListPendingLiveNowSessions).toHaveBeenCalled();
        expect(bindings.ClearPendingLiveNowSession).toHaveBeenCalledWith('bluesky-main');
        expect(bindings.ListDestinations).toHaveBeenCalled();
        expect(bindings.SaveDestination).toHaveBeenCalledWith(expect.objectContaining(destination));
        expect(bindings.DeleteDestination).toHaveBeenCalledWith('discord-main');
        expect(bindings.GetSettings).toHaveBeenCalled();
        expect(bindings.SaveSettings).toHaveBeenCalledWith(expect.objectContaining(settings));
        expect(bindings.TestDestinationConnection).toHaveBeenCalledWith(expect.objectContaining(destination));
    });
});
