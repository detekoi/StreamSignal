import type { ExecutionResult } from './execution';

export interface ActiveLiveNowSession {
    destinationID: string;
    destinationName: string;
    platform: string;
    accountIdentifier: string;
    credentialKey: string;
    streamURL: string;
    streamTitle: string;
    startedAt: string;
}

export type LiveNowRecoveryResult = ExecutionResult;
