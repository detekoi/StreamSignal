export type DestinationPlatform = 'discord' | 'bluesky' | 'mastodon';

export interface DestinationInput {
    id: string;
    platform: DestinationPlatform;
    name: string;
    enabled: boolean;
    template: string;
    configJSON: string;
    createdAt?: string;
    updatedAt?: string;
}

export interface DestinationFormState {
    id: string;
    platform: DestinationPlatform;
    name: string;
    enabled: boolean;
    template: string;
    createdAt?: string;
    updatedAt?: string;
    discordServerName: string;
    discordChannelName: string;
    discordWebhookKey: string;
    blueskyAccountIdentifier: string;
    blueskyCredentialKey: string;
    blueskyLiveStatusTemplate: string;
    mastodonAccountIdentifier: string;
    mastodonInstanceURL: string;
    mastodonCredentialKey: string;
}

interface DiscordConfig {
    serverName?: string;
    channelName?: string;
    webhookKey?: string;
}

interface BlueskyConfig {
    accountIdentifier?: string;
    credentialKey?: string;
    liveStatusTemplate?: string;
}

interface MastodonConfig {
    accountIdentifier?: string;
    instanceURL?: string;
    credentialKey?: string;
}

function parseConfigJSON<T>(configJSON: string): T {
    try {
        return JSON.parse(configJSON || '{}') as T;
    } catch {
        return {} as T;
    }
}

export function createEmptyDestinationForm(platform: DestinationPlatform = 'discord'): DestinationFormState {
    return {
        id: '',
        platform,
        name: '',
        enabled: true,
        template: '',
        createdAt: '',
        updatedAt: '',
        discordServerName: '',
        discordChannelName: '',
        discordWebhookKey: '',
        blueskyAccountIdentifier: '',
        blueskyCredentialKey: '',
        blueskyLiveStatusTemplate: '',
        mastodonAccountIdentifier: '',
        mastodonInstanceURL: '',
        mastodonCredentialKey: '',
    };
}

export function toDestinationFormState(destination: DestinationInput): DestinationFormState {
    const base = createEmptyDestinationForm(destination.platform);
    const next: DestinationFormState = {
        ...base,
        id: destination.id,
        platform: destination.platform,
        name: destination.name,
        enabled: destination.enabled,
        template: destination.template,
        createdAt: destination.createdAt ?? '',
        updatedAt: destination.updatedAt ?? '',
    };

    if (destination.platform === 'discord') {
        const config = parseConfigJSON<DiscordConfig>(destination.configJSON);
        next.discordServerName = config.serverName ?? '';
        next.discordChannelName = config.channelName ?? '';
        next.discordWebhookKey = config.webhookKey ?? '';
    }

    if (destination.platform === 'bluesky') {
        const config = parseConfigJSON<BlueskyConfig>(destination.configJSON);
        next.blueskyAccountIdentifier = config.accountIdentifier ?? '';
        next.blueskyCredentialKey = config.credentialKey ?? '';
        next.blueskyLiveStatusTemplate = config.liveStatusTemplate ?? '';
    }

    if (destination.platform === 'mastodon') {
        const config = parseConfigJSON<MastodonConfig>(destination.configJSON);
        next.mastodonAccountIdentifier = config.accountIdentifier ?? '';
        next.mastodonInstanceURL = config.instanceURL ?? '';
        next.mastodonCredentialKey = config.credentialKey ?? '';
    }

    return next;
}

export function toDestinationInput(form: DestinationFormState): DestinationInput {
    let configJSON = '{}';

    if (form.platform === 'discord') {
        configJSON = JSON.stringify({
            serverName: form.discordServerName,
            channelName: form.discordChannelName,
            webhookKey: form.discordWebhookKey,
        });
    }

    if (form.platform === 'bluesky') {
        configJSON = JSON.stringify({
            accountIdentifier: form.blueskyAccountIdentifier,
            credentialKey: form.blueskyCredentialKey,
            liveStatusTemplate: form.blueskyLiveStatusTemplate,
        });
    }

    if (form.platform === 'mastodon') {
        configJSON = JSON.stringify({
            accountIdentifier: form.mastodonAccountIdentifier,
            instanceURL: form.mastodonInstanceURL,
            credentialKey: form.mastodonCredentialKey,
        });
    }

    return {
        id: form.id,
        platform: form.platform,
        name: form.name,
        enabled: form.enabled,
        template: form.template,
        configJSON,
        createdAt: form.createdAt,
        updatedAt: form.updatedAt,
    };
}
