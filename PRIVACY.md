# Privacy Policy

**ArchiveTune Bot does not store message content.**

The bot performs deterministic tag and post management in Discord Forum Channels. Discord remains the single source of truth: the bot does not copy, retain, or analyze the content of your messages, posts, or conversations.

## What the bot does NOT do

- It does not store message content, post content, or chat history.
- It has no database and keeps no content history.
- It does not log or store message content, and it does not share it with third parties.
- It does not use message content for analytics or AI/ML training, and it does not use AI for moderation, classification, or content processing.
- It does not manage reactions or collect user data to build profiles.

## Message Content Intent and user control

The bot needs the Message Content Intent only for its prefix commands, such as `.solved`, `.accept`, `.dupe <link>`, `.reject <reason>`, `.help`, and `.ytd <URL>`. Reading message content is required to detect the command prefix and parse arguments such as a duplicate post link, a rejection reason, or a media URL. Moderator prefix commands must be typed inside Forum Channel posts; `.help` and `.ytd` are available to all members.

The bot reacts only to messages that start with the command prefix and ignores all other messages. In that sense, users can opt out simply by not issuing prefix commands — their ordinary messages are never processed. Message content is handled in memory and discarded immediately; it is never written to disk.

## Data the bot does store

The bot only reads and writes configuration in a local `config.yaml` file on the host running the bot. This file contains:

- The bot token and the guild / channel IDs required for operation.
- Tag names, guidelines, and other configuration options.

`config.yaml` is a local file kept outside version control (it is listed in `.gitignore`) and contains no user message content.

## Data processed temporarily

When you use `.ytd <URL>` or `/ytd url:<URL>`, the bot passes the media URL you provided to [yt-dlp](https://github.com/yt-dlp/yt-dlp) to download the media file. The URL itself is read from the command message, processed in memory, and discarded — it is not stored. The downloaded **media file** (not your message content) is then:

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
