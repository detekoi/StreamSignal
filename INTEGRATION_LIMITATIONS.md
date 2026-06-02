# Integration Limitations

This document records practical limits and constraints in StreamSignal's current platform integrations. It is meant to be a product and implementation reference, not an end-user troubleshooting guide.

## Discord

- StreamSignal currently uses incoming webhooks for Discord posting.
- A Discord webhook is tied to a specific channel and must already exist.
- In practice, this means Discord posting only works when:
  - you can create the webhook yourself, or
  - a server admin or moderator creates the webhook and gives you the URL.
- Simply being a member of a Discord server is not enough for automatic posting.
- Discord can generate its own native link preview when a stream URL appears in the message content, but StreamSignal cannot control that preview image.
- StreamSignal can also send an optional additional image through the webhook embed. Discord may show that image instead of its native link preview; the image is not a Twitch inline player.
- End Stream messages are configured per destination, so a Discord destination can leave End Stream posting disabled while other platforms still post or clear live state.
- StreamSignal does not automate normal user accounts and does not support self-bot style behavior.
- If broader Discord support is needed later, the legitimate expansion path is a real Discord app or bot installation flow with server authorization.

## Bluesky

- StreamSignal currently uses account identifier plus app password authentication.
- Bluesky `Live Now` support depends on Bluesky's current API and product behavior.
- StreamSignal sets and clears the dedicated `Live Now` status record. It does not modify normal profile text as part of the live workflow.
- StreamSignal can attach a stream-card thumbnail by fetching a direct HTTP(S) image URL or by uploading a local image selected in the destination form.
- StreamSignal can also attach an additional Bluesky image when no stream preview card is posted. If both are configured, the stream preview card takes priority.
- Uploaded local card images are resized in the frontend when needed, stored in local destination configuration as image data, and uploaded to Bluesky as a blob when posting.
- Bluesky image URLs are fetched only after public-address validation; localhost, private network, link-local, unspecified, and multicast targets are rejected.
- Bluesky images must fit Bluesky's current blob limits, so oversized images may be compressed or rejected.
- StreamSignal does not currently fetch a Twitch profile image or stream thumbnail automatically. If Twitch metadata does not provide the expected preview, configure a card thumbnail manually.
- If Bluesky changes the `Live Now` record contract or rollout behavior, the integration may need updates.

## Mastodon

- StreamSignal currently uses manual instance URL plus access token setup.
- Mastodon compatibility depends on the target instance supporting the expected API behavior.
- Because Mastodon is federated, behavior can vary slightly by instance policy, version, or token permissions.
- StreamSignal can attach one optional additional image by fetching a public HTTP(S) image URL or by uploading a local image selected in the destination form.
- Mastodon image URLs are fetched only after public-address validation; localhost, private network, link-local, unspecified, and multicast targets are rejected.
- Local Mastodon image uploads are resized in the frontend when needed, stored in local destination configuration as image data, uploaded to the instance media API, and attached to the status post.

## Cross-Platform Constraints

- StreamSignal is designed for automated outbound posting, not for reading back channel state from every platform.
- Some integrations require server, instance, or account-level approval outside the app before StreamSignal can post.
- Credential setup is currently guided manual entry plus connection testing, not full OAuth sign-in for every platform.
- Test destinations are configured as normal destinations with their own credentials, so test posts go only to the accounts or channels selected on Home.

## Why This Exists

These constraints are normal for third-party platform integrations, especially where platform safety, permissions, and anti-abuse rules are involved. This document exists so future UX, roadmap, and support decisions have a clear reference point.
