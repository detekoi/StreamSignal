import {
    ClearPendingLiveNowSession,
    DeleteDestination,
    DryRun,
    EndStream,
    ForceGoLive,
    GeneratePreview,
    GetDiagnostics,
    GetLogs,
    GoLive,
    GetOverview,
    GetSettings,
    ListPendingLiveNowSessions,
    ListDestinations,
    SaveDestination,
    SaveSettings,
} from '../../../wailsjs/go/main/App';
import type { AppOverview } from '../../types/app-overview';
import type { DestinationInput } from '../../types/destination';
import type { ExecutionSummary } from '../../types/execution';
import type { ActiveLiveNowSession, LiveNowRecoveryResult } from '../../types/live-now-recovery';
import type { LogEntry } from '../../types/log-entry';
import type { AnnouncementInput, PreviewItem } from '../../types/preview';
import type { AppSettings } from '../../types/settings';

export function getOverview(): Promise<AppOverview> {
    return GetOverview();
}

export function getLogs(): Promise<LogEntry[]> {
    return GetLogs();
}

export function getDiagnostics(): Promise<string> {
    return GetDiagnostics();
}

export function generatePreview(announcement: AnnouncementInput): Promise<PreviewItem[]> {
    return GeneratePreview(announcement);
}

export function dryRun(announcement: AnnouncementInput): Promise<ExecutionSummary> {
    return DryRun(announcement);
}

export function goLive(announcement: AnnouncementInput): Promise<ExecutionSummary> {
    return GoLive(announcement);
}

export function forceGoLive(announcement: AnnouncementInput): Promise<ExecutionSummary> {
    return ForceGoLive(announcement);
}

export function endStream(): Promise<ExecutionSummary> {
    return EndStream();
}

export function listPendingLiveNowSessions(): Promise<ActiveLiveNowSession[]> {
    return ListPendingLiveNowSessions();
}

export function clearPendingLiveNowSession(destinationID: string): Promise<LiveNowRecoveryResult> {
    return ClearPendingLiveNowSession(destinationID);
}

export function listDestinations(): Promise<DestinationInput[]> {
    return ListDestinations();
}

export function saveDestination(destination: DestinationInput): Promise<DestinationInput> {
    return SaveDestination(destination);
}

export function deleteDestination(id: string): Promise<void> {
    return DeleteDestination(id);
}

export function getSettings(): Promise<AppSettings> {
    return GetSettings();
}

export function saveSettings(settings: AppSettings): Promise<AppSettings> {
    return SaveSettings(settings);
}
