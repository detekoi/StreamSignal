# Integration Limitations

This document records practical limits and constraints in StreamSignal's current platform integrations. It is meant to be a product and implementation reference, not an end-user troubleshooting guide.

## Discord

- StreamSignal currently uses incoming webhooks for Discord posting.
- A Discord webhook is tied to a specific channel and must already exist.
- In practice, this means Discord posting only works when:
  - you can create the webhook yourself, or
  - a server admin or moderator creates the webhook and gives you the URL.
- Simply being a member of a Discord server is not enough for automatic posting.
- StreamSignal does not automate normal user accounts and does not support self-bot style behavior.
- If broader Discord support is needed later, the legitimate expansion path is a real Discord app or bot installation flow with server authorization.

## Bluesky

- StreamSignal currently uses account identifier plus app password authentication.
- Bluesky `Live Now` support depends on Bluesky's current API and product behavior.
- StreamSignal sets and clears the dedicated `Live Now` status record. It does not modify normal profile text as part of the live workflow.
- If Bluesky changes the `Live Now` record contract or rollout behavior, the integration may need updates.

## Mastodon

- StreamSignal currently uses manual instance URL plus access token setup.
- Mastodon compatibility depends on the target instance supporting the expected API behavior.
- Because Mastodon is federated, behavior can vary slightly by instance policy, version, or token permissions.

## Cross-Platform Constraints

- StreamSignal is designed for automated outbound posting, not for reading back channel state from every platform.
- Some integrations require server, instance, or account-level approval outside the app before StreamSignal can post.
- Credential setup is currently guided manual entry plus connection testing, not full OAuth sign-in for every platform.
- Test Mode helps route posts away from production destinations, but it still depends on valid test credentials for each platform.

## Why This Exists

These constraints are normal for third-party platform integrations, especially where platform safety, permissions, and anti-abuse rules are involved. This document exists so future UX, roadmap, and support decisions have a clear reference point.
