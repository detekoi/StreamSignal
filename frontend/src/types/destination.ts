export type DestinationPlatform = 'discord' | 'bluesky' | 'mastodon';
export type DestinationEnvironment = 'production' | 'test';

export const TEMPLATE_VARIABLES = [
    '{{stream_title}}',
    '{{stream_url}}',
    '{{category}}',
    '{{message}}',
    '{{hashtags}}',
    '{{date}}',
    '{{time}}',
    '{{platform}}',
] as const;

const TEMPLATE_VARIABLE_LABELS: Record<(typeof TEMPLATE_VARIABLES)[number], string> = {
    '{{stream_title}}': 'title',
    '{{stream_url}}': 'stream URL',
    '{{category}}': 'category',
    '{{message}}': 'message',
    '{{hashtags}}': 'hashtags',
    '{{date}}': 'date',
    '{{time}}': 'time',
    '{{platform}}': 'platform',
};

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
    environment: DestinationEnvironment;
    template: string;
    createdAt?: string;
    updatedAt?: string;
    discordServerName: string;
    discordChannelName: string;
    discordWebhookKey: string;
    blueskyAccountIdentifier: string;
    blueskyCredentialKey: string;
    blueskyLiveStatusTemplate: string;
    blueskyLiveNowDurationMinutes: string;
    blueskyCardThumbnailURL: string;
    blueskyCardThumbnailDataURL: string;
    mastodonAccountIdentifier: string;
    mastodonInstanceURL: string;
    mastodonCredentialKey: string;
}

interface DiscordConfig {
    serverName?: string;
    channelName?: string;
    webhookKey?: string;
    environment?: DestinationEnvironment;
}

interface BlueskyConfig {
    accountIdentifier?: string;
    credentialKey?: string;
    liveStatusTemplate?: string;
    liveNowDurationMinutes?: number;
    cardThumbnailURL?: string;
    cardThumbnailDataURL?: string;
    environment?: DestinationEnvironment;
}

interface MastodonConfig {
    accountIdentifier?: string;
    instanceURL?: string;
    credentialKey?: string;
    environment?: DestinationEnvironment;
}

function parseConfigJSON<T>(configJSON: string): T {
    try {
        return JSON.parse(configJSON || '{}') as T;
    } catch {
        return {} as T;
    }
}

function parseEnvironment(value?: string): DestinationEnvironment {
    return value === 'test' ? 'test' : 'production';
}

export function defaultTemplateForPlatform(platform: DestinationPlatform): string {
    switch (platform) {
        case 'discord':
            return '{{stream_title}}\n{{stream_url}}\n{{message}}\n{{hashtags}}';
        case 'bluesky':
            return '{{stream_title}}\n{{stream_url}}\n{{hashtags}}';
        case 'mastodon':
            return '{{stream_title}}\n{{stream_url}}\n{{message}}\n{{hashtags}}';
        default:
            return '{{stream_title}}';
    }
}

export function usedTemplateVariables(template: string): string[] {
    return TEMPLATE_VARIABLES.filter((token) => template.includes(token)).map((token) => TEMPLATE_VARIABLE_LABELS[token]);
}

export function createEmptyDestinationForm(platform: DestinationPlatform = 'discord'): DestinationFormState {
    return {
        id: '',
        platform,
        name: '',
        environment: 'production',
        template: defaultTemplateForPlatform(platform),
        createdAt: '',
        updatedAt: '',
        discordServerName: '',
        discordChannelName: '',
        discordWebhookKey: '',
        blueskyAccountIdentifier: '',
        blueskyCredentialKey: '',
        blueskyLiveStatusTemplate: '',
        blueskyLiveNowDurationMinutes: '120',
        blueskyCardThumbnailURL: '',
        blueskyCardThumbnailDataURL: '',
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
        template: destination.template,
        createdAt: destination.createdAt ?? '',
        updatedAt: destination.updatedAt ?? '',
    };

    if (destination.platform === 'discord') {
        const config = parseConfigJSON<DiscordConfig>(destination.configJSON);
        next.environment = parseEnvironment(config.environment);
        next.discordServerName = config.serverName ?? '';
        next.discordChannelName = config.channelName ?? '';
        next.discordWebhookKey = config.webhookKey ?? '';
    }

    if (destination.platform === 'bluesky') {
        const config = parseConfigJSON<BlueskyConfig>(destination.configJSON);
        next.environment = parseEnvironment(config.environment);
        next.blueskyAccountIdentifier = config.accountIdentifier ?? '';
        next.blueskyCredentialKey = config.credentialKey ?? '';
        next.blueskyLiveStatusTemplate = config.liveStatusTemplate ?? '';
        next.blueskyLiveNowDurationMinutes = String(config.liveNowDurationMinutes ?? 120);
        next.blueskyCardThumbnailURL = config.cardThumbnailURL ?? '';
        next.blueskyCardThumbnailDataURL = config.cardThumbnailDataURL ?? '';
    }

    if (destination.platform === 'mastodon') {
        const config = parseConfigJSON<MastodonConfig>(destination.configJSON);
        next.environment = parseEnvironment(config.environment);
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
            environment: form.environment,
            serverName: form.discordServerName,
            channelName: form.discordChannelName,
            webhookKey: form.discordWebhookKey,
        });
    }

    if (form.platform === 'bluesky') {
        configJSON = JSON.stringify({
            environment: form.environment,
            accountIdentifier: form.blueskyAccountIdentifier,
            credentialKey: form.blueskyCredentialKey,
            liveStatusTemplate: form.blueskyLiveStatusTemplate,
            liveNowDurationMinutes: Number.parseInt(form.blueskyLiveNowDurationMinutes, 10) || 120,
            cardThumbnailURL: form.blueskyCardThumbnailURL,
            cardThumbnailDataURL: form.blueskyCardThumbnailDataURL,
        });
    }

    if (form.platform === 'mastodon') {
        configJSON = JSON.stringify({
            environment: form.environment,
            accountIdentifier: form.mastodonAccountIdentifier,
            instanceURL: form.mastodonInstanceURL,
            credentialKey: form.mastodonCredentialKey,
        });
    }

    return {
        id: form.id,
        platform: form.platform,
        name: form.name,
        enabled: true,
        template: form.template,
        configJSON,
        createdAt: form.createdAt,
        updatedAt: form.updatedAt,
    };
}
