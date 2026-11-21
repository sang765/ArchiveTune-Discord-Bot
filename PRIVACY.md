# Privacy Policy

**ArchiveTune Bot does not store message content — and it no longer reads it.**

The bot performs deterministic tag and post management in Discord Forum Channels. Discord remains the single source of truth: the bot does not copy, retain, or analyze the content of your messages, posts, or conversations.

## What the bot does NOT do

- It does not request or use the **Message Content Intent**. It never reads message bodies.
- It does not store message content, post content, or chat history.
- It has no database and keeps no content history.
- It does not log or store message content, and it does not share it with third parties.
- It does not use message content for analytics or AI/ML training, and it does not use AI for moderation, classification, or content processing.
- It does not manage reactions or collect user data to build profiles.

## How the bot is triggered

Every command is a Discord app (slash) command or an interaction component (buttons and select menus). Moderator workflows — such as `/solved`, `/accept`, `/dupe`, and `/reject` — were migrated from prefix commands to slash commands with identical behavior. Users simply choose whether to invoke a slash command; ordinary messages are never read or processed by the bot.

For automatic moderation, the bot uses only Discord `ThreadCreate` and `MessageDelete` gateway events (to tag new suggestion posts and clean up deleted posts). These events provide IDs and metadata, not message content.

## Data the bot does store

The bot only reads and writes configuration in a local `config.yaml` file on the host running the bot. This file contains:

- The bot token and the guild / channel IDs required for operation.
- Tag names, guidelines, and other configuration options.

`config.yaml` is a local file kept outside version control (it is listed in `.gitignore`) and contains no user message content.

## Data processed temporarily

When you use `/ytd`, the media URL you provide as a command option is passed to [yt-dlp](https://github.com/yt-dlp/yt-dlp) to download the media file. The URL is supplied explicitly by you through a slash-command option — it is not read from a message body — and is processed in memory and discarded. The downloaded **media file** (not your message content) is then:

1. Downloaded into a temporary directory created by the operating system.
2. Uploaded to [temp.sh](https://temp.sh/), a third-party temporary hosting service (currently limited to 4GB and expiring after three days), and returned to you as a temporary download link.
3. Removed along with the entire temporary directory immediately after processing, whether the job succeeds or fails.

The bot keeps no copy of the media on disk once the command finishes. Uploading the downloaded media to a temporary host does not mean message content is stored off-platform. Links from temp.sh are temporary and will expire; download the file before then.

## Logs

The bot may write technical operational logs (startup errors, API errors, command status) for maintenance purposes. Logs are not used to store message content. Avoid sharing log files publicly, as they may contain channel IDs or server technical details.

## Discord access

The bot uses the Discord Gateway and Discord REST API to perform only the configured operations (syncing tags, adding/removing tags, locking posts). It does not read or store content beyond what is necessary for those operations.

## Contact

If you have privacy questions or want configuration related to your server removed, contact the bot administrator.
