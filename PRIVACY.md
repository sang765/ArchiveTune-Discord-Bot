# Privacy Policy

**ArchiveTune Bot does not store message content.**

The bot performs deterministic tag and post management in Discord Forum Channels. Discord remains the single source of truth: the bot does not copy, retain, or analyze the content of your messages, posts, or conversations.

## What the bot does NOT do

- It does not store message content, post content, or chat history.
- It has no database and keeps no content history.
- It does not use AI for moderation, classification, or content processing.
- It does not manage reactions or collect user data to build profiles.

## Data the bot does store

The bot only reads and writes configuration in a local `config.yaml` file on the host running the bot. This file contains:

- The bot token and the guild / channel IDs required for operation.
- Tag names, guidelines, and other configuration options.

`config.yaml` is a local file kept outside version control (it is listed in `.gitignore`) and contains no user message content.

## Data processed temporarily

When you use a media download command, the bot will:

1. Download the media file into a temporary directory created by the operating system.
2. Upload the file to [temp.sh](https://temp.sh/), a third-party temporary hosting service (currently limited to 4GB and expiring after three days), and return a temporary download link.
3. **Delete the entire temporary directory immediately after processing**, whether the job succeeds or fails.

The bot keeps no copy of the media on disk once the command finishes. Links from temp.sh are temporary and will expire; download the file before then.

## Logs

The bot may write technical operational logs (startup errors, API errors, command status) for maintenance purposes. Logs are not used to store message content. Avoid sharing log files publicly, as they may contain channel IDs or server technical details.

## Discord access

The bot uses the Discord Gateway and Discord REST API to perform only the configured operations (syncing tags, adding/removing tags, locking posts). It does not read or store content beyond what is necessary for those operations.

## Contact

If you have privacy questions or want configuration related to your server removed, contact the bot administrator.
