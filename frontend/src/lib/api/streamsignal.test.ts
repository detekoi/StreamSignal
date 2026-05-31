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
    GetOverview: vi.fn(),
    GetSettings: vi.fn(),
    GoLive: vi.fn(),
    ListPendingLiveNowSessions: vi.fn(),
    ListDestinations: vi.fn(),
    SaveDestination: vi.fn(),
    SaveSettings: vi.fn(),
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
    getOverview,
    getSettings,
    goLive,
    listDestinations,
    listPendingLiveNowSessions,
    saveDestination,
    saveSettings,
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

        vi.mocked(bindings.GetOverview).mockResolvedValue({ productName: 'StreamSignal' } as never);
        vi.mocked(bindings.GetLogs).mockResolvedValue([] as never);
        vi.mocked(bindings.GetDiagnostics).mockResolvedValue('diag' as never);
        vi.mocked(bindings.GeneratePreview).mockResolvedValue([] as never);
        vi.mocked(bindings.DryRun).mockResolvedValue({ mode: 'dry_run' } as never);
        vi.mocked(bindings.GoLive).mockResolvedValue({ mode: 'go_live' } as never);
        vi.mocked(bindings.ForceGoLive).mockResolvedValue({ mode: 'go_live' } as never);
        vi.mocked(bindings.EndStream).mockResolvedValue({ mode: 'end_stream' } as never);
        vi.mocked(bindings.ListPendingLiveNowSessions).mockResolvedValue([] as never);
        vi.mocked(bindings.ClearPendingLiveNowSession).mockResolvedValue({} as never);
        vi.mocked(bindings.ListDestinations).mockResolvedValue([] as never);
        vi.mocked(bindings.SaveDestination).mockResolvedValue(destination as never);
        vi.mocked(bindings.DeleteDestination).mockResolvedValue(undefined as never);
        vi.mocked(bindings.GetSettings).mockResolvedValue(settings as never);
        vi.mocked(bindings.SaveSettings).mockResolvedValue(settings as never);

        await getOverview();
        await getLogs();
        await getDiagnostics();
        await generatePreview(announcement);
        await dryRun(announcement);
        await goLive(announcement);
        await forceGoLive(announcement);
        await endStream();
        await listPendingLiveNowSessions();
        await clearPendingLiveNowSession('bluesky-main');
        await listDestinations();
        await saveDestination(destination);
        await deleteDestination('discord-main');
        await getSettings();
        await saveSettings(settings);

        expect(bindings.GetOverview).toHaveBeenCalled();
        expect(bindings.GetLogs).toHaveBeenCalled();
        expect(bindings.GetDiagnostics).toHaveBeenCalled();
        expect(bindings.GeneratePreview).toHaveBeenCalledWith(announcement);
        expect(bindings.DryRun).toHaveBeenCalledWith(announcement);
        expect(bindings.GoLive).toHaveBeenCalledWith(announcement);
        expect(bindings.ForceGoLive).toHaveBeenCalledWith(announcement);
        expect(bindings.EndStream).toHaveBeenCalled();
        expect(bindings.ListPendingLiveNowSessions).toHaveBeenCalled();
        expect(bindings.ClearPendingLiveNowSession).toHaveBeenCalledWith('bluesky-main');
        expect(bindings.ListDestinations).toHaveBeenCalled();
        expect(bindings.SaveDestination).toHaveBeenCalledWith(destination);
        expect(bindings.DeleteDestination).toHaveBeenCalledWith('discord-main');
        expect(bindings.GetSettings).toHaveBeenCalled();
        expect(bindings.SaveSettings).toHaveBeenCalledWith(settings);
    });
});
