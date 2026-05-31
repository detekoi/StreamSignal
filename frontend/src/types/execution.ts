export interface ExecutionResult {
    destinationID: string;
    destinationName: string;
    platform: string;
    state: 'SUCCESS' | 'FAILED' | 'SKIPPED' | 'VALIDATION_ERROR';
    message: string;
    content: string;
}

export interface ExecutionSummary {
    mode: 'dry_run' | 'go_live' | 'end_stream';
    status: 'SUCCESS' | 'WARNING' | 'PARTIAL';
    testModeActive: boolean;
    results: ExecutionResult[];
    totalCount: number;
    successCount: number;
    failedCount: number;
    skippedCount: number;
    validationErrorCount: number;
    requiresDuplicateConfirmation: boolean;
    duplicateWarningMessage: string;
}
