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
            configJSON: '{"serverName":"My Server","channelName":"go-live","webhookKey":"discord/main"}',
            createdAt: '',
            updatedAt: '',
        };

        const form = toDestinationFormState(destination);

        expect(form.discordServerName).toBe('My Server');
        expect(form.discordChannelName).toBe('go-live');
        expect(form.discordWebhookKey).toBe('discord/main');
    });

    it('serializes a Bluesky form state into destination config json', () => {
        const form = createEmptyDestinationForm('bluesky');
        form.name = 'Main Bluesky';
        form.template = '{{stream_title}}';
        form.blueskyAccountIdentifier = 'don.test';
        form.blueskyCredentialKey = 'bluesky/main';
        form.blueskyLiveStatusTemplate = 'LIVE {{stream_title}}';

        const destination = toDestinationInput(form);

        expect(destination.platform).toBe('bluesky');
        expect(destination.configJSON).toBe(
            '{"accountIdentifier":"don.test","credentialKey":"bluesky/main","liveStatusTemplate":"LIVE {{stream_title}}"}',
        );
    });

    it('creates a Bluesky form state from destination config json', () => {
        const destination: DestinationInput = {
            id: 'bluesky-main',
            platform: 'bluesky',
            name: 'Main Bluesky',
            enabled: true,
            template: '{{stream_title}}',
            configJSON: '{"accountIdentifier":"don.test","credentialKey":"bluesky/main","liveStatusTemplate":"LIVE {{stream_title}}"}',
            createdAt: '',
            updatedAt: '',
        };

        const form = toDestinationFormState(destination);

        expect(form.blueskyAccountIdentifier).toBe('don.test');
        expect(form.blueskyCredentialKey).toBe('bluesky/main');
        expect(form.blueskyLiveStatusTemplate).toBe('LIVE {{stream_title}}');
    });

    it('serializes a Mastodon form state into destination config json', () => {
        const form = createEmptyDestinationForm('mastodon');
        form.name = 'Main Mastodon';
        form.template = '{{stream_title}}';
        form.mastodonAccountIdentifier = '@don@example.social';
        form.mastodonInstanceURL = 'https://mastodon.social';
        form.mastodonCredentialKey = 'mastodon/main';

        const destination = toDestinationInput(form);

        expect(destination.platform).toBe('mastodon');
        expect(destination.configJSON).toBe(
            '{"accountIdentifier":"@don@example.social","instanceURL":"https://mastodon.social","credentialKey":"mastodon/main"}',
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
