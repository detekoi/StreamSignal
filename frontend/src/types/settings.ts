export interface AppSettings {
    testModeEnabled: boolean;
    testDiscordWebhookKey: string;
    testBlueskyAccountIdentifier: string;
    testBlueskyCredentialKey: string;
    testMastodonCredentialKey: string;
    testMastodonInstanceURL: string;
    defaultStreamURL: string;
    defaultHashtags: string;
    duplicateProtectionEnabled: boolean;
    duplicateWindowMinutes: number;
    endStreamPostEnabled: boolean;
    endStreamTemplate: string;
}
