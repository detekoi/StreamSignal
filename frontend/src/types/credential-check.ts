export type CredentialCheckState = 'SUCCESS' | 'FAILED' | 'VALIDATION_ERROR';

export interface CredentialCheckResult {
    platform: 'discord' | 'bluesky' | 'mastodon';
    state: CredentialCheckState;
    message: string;
}
