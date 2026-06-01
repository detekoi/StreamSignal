/// <reference types="vitest/globals" />
import {
    createEmptyDestinationForm,
    defaultTemplateForPlatform,
    toDestinationFormState,
    toDestinationInput,
    usedTemplateVariables,
    type DestinationInput,
} from './destination';

describe('destination mappings', () => {
    it('creates a Discord form state from destination config json', () => {
        const destination: DestinationInput = {
            id: 'discord-main',
            platform: 'discord',
            name: 'Main Discord',
            enabled: true,
            template: '{{stream_title}}',
            configJSON: '{"serverName":"My Server","channelName":"go-live","webhookKey":"discord/main","cardThumbnailURL":"https://example.com/card.png","cardThumbnailDataURL":"data:image/png;base64,abc","endStreamEnabled":true,"endStreamTemplate":"Bye {{stream_title}}"}',
            createdAt: '',
            updatedAt: '',
        };

        const form = toDestinationFormState(destination);

        expect(form.discordServerName).toBe('My Server');
        expect(form.discordChannelName).toBe('go-live');
        expect(form.discordWebhookKey).toBe('discord/main');
        expect(form.discordCardThumbnailURL).toBe('https://example.com/card.png');
        expect(form.discordCardThumbnailDataURL).toBe('data:image/png;base64,abc');
        expect(form.endStreamEnabled).toBe(true);
        expect(form.endStreamTemplate).toBe('Bye {{stream_title}}');
        expect(form.environment).toBe('production');
    });

    it('serializes a Discord form state into destination config json', () => {
        const form = createEmptyDestinationForm('discord');
        form.name = 'Main Discord';
        form.template = '{{stream_title}}';
        form.discordServerName = 'My Server';
        form.discordChannelName = 'go-live';
        form.discordWebhookKey = 'discord/main';
        form.discordCardThumbnailURL = 'https://example.com/card.png';
        form.discordCardThumbnailDataURL = 'data:image/png;base64,abc';
        form.endStreamEnabled = true;
        form.endStreamTemplate = 'Bye {{stream_title}}';

        const destination = toDestinationInput(form);

        expect(destination.platform).toBe('discord');
        expect(destination.configJSON).toBe(
            '{"environment":"production","serverName":"My Server","channelName":"go-live","webhookKey":"discord/main","cardThumbnailURL":"https://example.com/card.png","cardThumbnailDataURL":"data:image/png;base64,abc","endStreamEnabled":true,"endStreamTemplate":"Bye {{stream_title}}"}',
        );
    });

    it('serializes a Bluesky form state into destination config json', () => {
        const form = createEmptyDestinationForm('bluesky');
        form.name = 'Main Bluesky';
        form.template = '{{stream_title}}';
        form.blueskyAccountIdentifier = 'don.test';
        form.blueskyCredentialKey = 'bluesky/main';
        form.blueskyLiveStatusTemplate = 'LIVE {{stream_title}}';
        form.blueskyLiveNowDurationMinutes = '90';
        form.blueskyCardThumbnailURL = 'https://example.com/avatar.png';
        form.blueskyCardThumbnailDataURL = 'data:image/png;base64,abc';
        form.endStreamEnabled = true;
        form.endStreamTemplate = 'Bye {{stream_title}}';

        const destination = toDestinationInput(form);

        expect(destination.platform).toBe('bluesky');
        expect(destination.configJSON).toBe(
            '{"environment":"production","accountIdentifier":"don.test","credentialKey":"bluesky/main","liveStatusTemplate":"LIVE {{stream_title}}","liveNowDurationMinutes":90,"cardThumbnailURL":"https://example.com/avatar.png","cardThumbnailDataURL":"data:image/png;base64,abc","endStreamEnabled":true,"endStreamTemplate":"Bye {{stream_title}}"}',
        );
    });

    it('creates a Bluesky form state from destination config json', () => {
        const destination: DestinationInput = {
            id: 'bluesky-main',
            platform: 'bluesky',
            name: 'Main Bluesky',
            enabled: true,
            template: '{{stream_title}}',
            configJSON: '{"environment":"test","accountIdentifier":"don.test","credentialKey":"bluesky/main","liveStatusTemplate":"LIVE {{stream_title}}","liveNowDurationMinutes":45,"cardThumbnailURL":"https://example.com/avatar.png","cardThumbnailDataURL":"data:image/png;base64,abc","endStreamEnabled":true,"endStreamTemplate":"Bye {{stream_title}}"}',
            createdAt: '',
            updatedAt: '',
        };

        const form = toDestinationFormState(destination);

        expect(form.blueskyAccountIdentifier).toBe('don.test');
        expect(form.blueskyCredentialKey).toBe('bluesky/main');
        expect(form.blueskyLiveStatusTemplate).toBe('LIVE {{stream_title}}');
        expect(form.blueskyLiveNowDurationMinutes).toBe('45');
        expect(form.blueskyCardThumbnailURL).toBe('https://example.com/avatar.png');
        expect(form.blueskyCardThumbnailDataURL).toBe('data:image/png;base64,abc');
        expect(form.endStreamEnabled).toBe(true);
        expect(form.endStreamTemplate).toBe('Bye {{stream_title}}');
        expect(form.environment).toBe('test');
    });

    it('serializes a Mastodon form state into destination config json', () => {
        const form = createEmptyDestinationForm('mastodon');
        form.name = 'Main Mastodon';
        form.template = '{{stream_title}}';
        form.mastodonAccountIdentifier = '@don@example.social';
        form.mastodonInstanceURL = 'https://mastodon.social';
        form.mastodonCredentialKey = 'mastodon/main';
        form.endStreamEnabled = true;
        form.endStreamTemplate = 'Bye {{stream_title}}';

        const destination = toDestinationInput(form);

        expect(destination.platform).toBe('mastodon');
        expect(destination.configJSON).toBe(
            '{"environment":"production","accountIdentifier":"@don@example.social","instanceURL":"https://mastodon.social","credentialKey":"mastodon/main","endStreamEnabled":true,"endStreamTemplate":"Bye {{stream_title}}"}',
        );
    });

    it('falls back safely when config json is invalid', () => {
        const form = toDestinationFormState({
            id: 'mastodon-main',
            platform: 'mastodon',
            name: 'Main Mastodon',
            enabled: true,
            template: '{{stream_title}}',
            configJSON: '{invalid-json',
            createdAt: '',
            updatedAt: '',
        });

        expect(form.mastodonAccountIdentifier).toBe('');
        expect(form.mastodonInstanceURL).toBe('');
        expect(form.mastodonCredentialKey).toBe('');
    });

    it('creates platform starter templates for new destinations', () => {
        expect(defaultTemplateForPlatform('discord')).toContain('{{stream_url}}');
        expect(createEmptyDestinationForm('bluesky').template).toBe('{{stream_title}}\n{{stream_url}}\n{{hashtags}}');
    });

    it('summarizes which announcement fields a template uses', () => {
        expect(usedTemplateVariables('{{stream_title}}\n{{stream_url}}\n{{hashtags}}')).toEqual(['title', 'stream URL', 'hashtags']);
        expect(usedTemplateVariables('plain text only')).toEqual([]);
    });
});
