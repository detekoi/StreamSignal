import {
    ClearPendingLiveNowSession,
    DeleteDestination,
    EndStream,
    ForceGoLive,
    GeneratePreview,
    GetDiagnostics,
    GetLogs,
    GoLive,
    GetSettings,
    ListPendingLiveNowSessions,
    ListDestinations,
    SaveDestination,
    SaveSettings,
    TestDestinationConnection,
} from '../../../wailsjs/go/main/App';
import { domain } from '../../../wailsjs/go/models';
import type { CredentialCheckResult } from '../../types/credential-check';
import type { DestinationInput, DestinationPlatform } from '../../types/destination';
import type { ExecutionResult, ExecutionSummary } from '../../types/execution';
import type { ActiveLiveNowSession, LiveNowRecoveryResult } from '../../types/live-now-recovery';
import type { LogEntry } from '../../types/log-entry';
import type { AnnouncementInput, PreviewItem } from '../../types/preview';
import type { AppSettings } from '../../types/settings';

function toAnnouncementModel(announcement: AnnouncementInput, destinationIDs: string[] = []): domain.Announcement {
    return {
        ...domain.Announcement.createFrom(announcement),
        destinationIDs,
    } as unknown as domain.Announcement;
}

function asArray<T>(value: T[] | null | undefined): T[] {
    return Array.isArray(value) ? value : [];
}

function toDestinationModel(destination: DestinationInput): domain.Destination {
    const { createdAt, updatedAt, ...rest } = destination;
    return domain.Destination.createFrom({
        ...rest,
        ...(createdAt ? { createdAt } : {}),
        ...(updatedAt ? { updatedAt } : {}),
    });
}

export function errorMessage(err: unknown, fallback: string): string {
    if (err instanceof Error) {
        return err.message;
    }
    if (typeof err === 'string') {
        return err;
    }
    return fallback;
}

function toExecutionResult(result: domain.ExecutionResult): ExecutionResult {
    return {
        destinationID: result.destinationID,
        destinationName: result.destinationName,
        platform: result.platform,
        state: result.state as ExecutionResult['state'],
        message: result.message,
        content: result.content,
    };
}

function toExecutionSummary(summary: domain.ExecutionSummary): ExecutionSummary {
    return {
        mode: summary.mode as ExecutionSummary['mode'],
        status: summary.status as ExecutionSummary['status'],
        testModeActive: summary.testModeActive,
        results: asArray(summary.results).map(toExecutionResult),
        totalCount: summary.totalCount,
        successCount: summary.successCount,
        failedCount: summary.failedCount,
        skippedCount: summary.skippedCount,
        validationErrorCount: summary.validationErrorCount,
        requiresDuplicateConfirmation: summary.requiresDuplicateConfirmation,
        duplicateWarningMessage: summary.duplicateWarningMessage,
    };
}

function toDestinationInput(destination: domain.Destination): DestinationInput {
    return {
        id: destination.id,
        platform: destination.platform as DestinationPlatform,
        name: destination.name,
        enabled: destination.enabled,
        template: destination.template,
        configJSON: destination.configJSON,
        createdAt: destination.createdAt ? String(destination.createdAt) : '',
        updatedAt: destination.updatedAt ? String(destination.updatedAt) : '',
    };
}

function toCredentialCheckResult(result: domain.CredentialCheckResult): CredentialCheckResult {
    return {
        platform: result.platform as CredentialCheckResult['platform'],
        state: result.state as CredentialCheckResult['state'],
        message: result.message,
    };
}

function toActiveLiveNowSession(session: domain.ActiveLiveNowSession): ActiveLiveNowSession {
    return {
        destinationID: session.destinationID,
        destinationName: session.destinationName,
        platform: session.platform,
        accountIdentifier: session.accountIdentifier,
        credentialKey: session.credentialKey,
        streamURL: session.streamURL,
        streamTitle: session.streamTitle,
        startedAt: session.startedAt ? String(session.startedAt) : '',
    };
}

function toLogEntry(entry: domain.LogEntry): LogEntry {
    return {
        timestamp: entry.timestamp ? String(entry.timestamp) : '',
        destination: entry.destination,
        action: entry.action,
        status: entry.status,
        message: entry.message,
    };
}

function toPreviewItem(item: domain.PreviewItem): PreviewItem {
    return {
        destinationID: item.destinationID,
        destinationName: item.destinationName,
        platform: item.platform,
        content: item.content,
        characterCount: item.characterCount,
        validationState: item.validationState,
        validationNotes: asArray(item.validationNotes),
    };
}

function toSettingsModel(settings: AppSettings): domain.AppSettings {
    return domain.AppSettings.createFrom(settings);
}

export function getLogs(): Promise<LogEntry[]> {
    return GetLogs().then((entries: domain.LogEntry[] | null) => asArray(entries).map(toLogEntry));
}

export function getDiagnostics(): Promise<string> {
    return GetDiagnostics();
}

export function generatePreview(announcement: AnnouncementInput, destinationIDs: string[] = []): Promise<PreviewItem[]> {
    return GeneratePreview(toAnnouncementModel(announcement, destinationIDs)).then((items: domain.PreviewItem[] | null) => asArray(items).map(toPreviewItem));
}

export function goLive(announcement: AnnouncementInput, destinationIDs: string[] = []): Promise<ExecutionSummary> {
    return GoLive(toAnnouncementModel(announcement, destinationIDs)).then(toExecutionSummary);
}

export function forceGoLive(announcement: AnnouncementInput, destinationIDs: string[] = []): Promise<ExecutionSummary> {
    return ForceGoLive(toAnnouncementModel(announcement, destinationIDs)).then(toExecutionSummary);
}

export function endStream(): Promise<ExecutionSummary> {
    return EndStream().then(toExecutionSummary);
}

export function listPendingLiveNowSessions(): Promise<ActiveLiveNowSession[]> {
    return ListPendingLiveNowSessions().then((sessions: domain.ActiveLiveNowSession[] | null) => asArray(sessions).map(toActiveLiveNowSession));
}

export function clearPendingLiveNowSession(destinationID: string): Promise<LiveNowRecoveryResult> {
    return ClearPendingLiveNowSession(destinationID).then(toExecutionResult);
}

export function listDestinations(): Promise<DestinationInput[]> {
    return ListDestinations().then((destinations: domain.Destination[] | null) => asArray(destinations).map(toDestinationInput));
}

export function saveDestination(destination: DestinationInput): Promise<DestinationInput> {
    return SaveDestination(toDestinationModel(destination)).then(toDestinationInput);
}

export function testDestinationConnection(destination: DestinationInput): Promise<CredentialCheckResult> {
    return TestDestinationConnection(toDestinationModel(destination)).then(toCredentialCheckResult);
}

export function deleteDestination(id: string): Promise<void> {
    return DeleteDestination(id);
}

export function getSettings(): Promise<AppSettings> {
    return GetSettings().then((settings: domain.AppSettings) => ({
        testModeEnabled: settings.testModeEnabled,
        testDiscordWebhookKey: settings.testDiscordWebhookKey,
        testBlueskyAccountIdentifier: settings.testBlueskyAccountIdentifier,
        testBlueskyCredentialKey: settings.testBlueskyCredentialKey,
        testMastodonCredentialKey: settings.testMastodonCredentialKey,
        testMastodonInstanceURL: settings.testMastodonInstanceURL,
        defaultStreamURL: settings.defaultStreamURL,
        defaultHashtags: settings.defaultHashtags,
        duplicateProtectionEnabled: settings.duplicateProtectionEnabled,
        duplicateWindowMinutes: settings.duplicateWindowMinutes,
        endStreamPostEnabled: settings.endStreamPostEnabled,
        endStreamTemplate: settings.endStreamTemplate,
    }));
}

export function saveSettings(settings: AppSettings): Promise<AppSettings> {
    return SaveSettings(toSettingsModel(settings)).then((saved: domain.AppSettings) => ({
        testModeEnabled: saved.testModeEnabled,
        testDiscordWebhookKey: saved.testDiscordWebhookKey,
        testBlueskyAccountIdentifier: saved.testBlueskyAccountIdentifier,
        testBlueskyCredentialKey: saved.testBlueskyCredentialKey,
        testMastodonCredentialKey: saved.testMastodonCredentialKey,
        testMastodonInstanceURL: saved.testMastodonInstanceURL,
        defaultStreamURL: saved.defaultStreamURL,
        defaultHashtags: saved.defaultHashtags,
        duplicateProtectionEnabled: saved.duplicateProtectionEnabled,
        duplicateWindowMinutes: saved.duplicateWindowMinutes,
        endStreamPostEnabled: saved.endStreamPostEnabled,
        endStreamTemplate: saved.endStreamTemplate,
    }));
}
