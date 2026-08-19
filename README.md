# Discord Forum Tag Manager Bot

[English](README.md) · [日本語](README.ja.md) · [Tiếng Việt](README.vi.md)

A focused Discord bot written in **Go** for managing tags and posts in Discord Forum Channels. The bot has no database and does not process content with AI. Discord remains the source of truth, while `config.yaml` stores channel and bot settings.

Discord represents a Forum post as a thread. Available Forum tags are stored on the parent channel, while the tags applied to an individual post are stored on the thread.[1] [2] This bot therefore uses the Gateway for realtime events and the Discord REST API for deterministic post and tag operations.

## Current scope

| Area | Behavior |
| --- | --- |
| Forum synchronization | Synchronizes guidelines, configured tags, and the `Require Tags when posting` flag. |
| Automatic suggestion tagging | Adds `Maybe` to every new untagged post in the configured suggestion channel. |
| Tag management | Adds, removes, or replaces tags on managed posts. |
| Post management | Renames, archives, unarchives, locks, and unlocks posts where the command requires it. |
| Moderation workflows | Provides status commands for issues and suggestions, including author mentions and rejection reasons. |
| Deployment | Supports a Go binary, Docker, or systemd. Secrets are read from a local configuration file and are not committed. |

The bot intentionally does not provide AI moderation, reaction management, history storage, or content-based automatic classification. Its boundary is deterministic Forum tag and post management.

## Managed channels

| Channel | ID | Special behavior |
| --- | --- | --- |
| `issues` | `1498327801923637439` | `.solved`, `.false`, and `.false-report` are restricted to this Forum Channel. Successful `.solved` and `.false` operations mention the post author. |
| `suggestion` | `1498328044635422790` | New untagged posts automatically receive `Maybe`. `/fix-suggestion`, `.dupe`, `.done`, `.in-progress`, `.exist`, `.reject`, `.tba`, and `.tbd` operate here. |

## Slash commands

| Command | Usage | Behavior |
| --- | --- | --- |
| `/forum-sync channel:<Forum Channel>` | Run after changing `config.yaml`. | Synchronizes guidelines, tags, and the required-tag flag. |
| `/fix-suggestion` | Moderator command. | Scans active and archived accessible suggestion posts and adds `Maybe` to posts with no applied tag. |
| `/tag-add tag:<name>` | Run in a post, or provide `post_id`. | Adds a configured tag without replacing existing tags. |
| `/tag-remove tag:<name>` | Run in a post, or provide `post_id`. | Removes a configured tag. |
| `/post-rename name:<new name>` | Run in a post, or provide `post_id`. | Renames a managed post. |
| `/post-state state:<open\|close\|lock\|unlock>` | Run in a post, or provide `post_id`. | Changes the archive or lock state of a managed post. |

## Prefix commands

Prefix commands are typed as normal messages. They require the **Message Content Intent** and moderator access.

### Issue commands

| Command | Behavior |
| --- | --- |
| `.solved` | In `issues`, replaces the current workflow tag with `Solved`, locks the post, and renames it to `[SOLVED] <old name>`. The bot mentions the author. |
| `.false` | In `issues`, replaces the current workflow tag with `False report`, locks the post, and renames it to `[FALSE REPORT] <old name>`. The bot mentions the author. |
| `.false-report` | Full-name alias for the `False report` workflow in `issues`. |

### Suggestion commands

| Command | Usage | Behavior |
| --- | --- | --- |
| `.dupe <post link or message link>` | Requires a Discord post/message link. | Replaces all existing tags with `Duplicate`, closes and locks the target suggestion post, renames it to `[DUPLICATE] <old name>`, and mentions the target author. |
| `.done` | Run directly inside the current suggestion post. | Replaces all existing tags with `Done`, closes and locks the post, and renames it to `[DONE] <old name>`. |
| `.in-progress` | Run directly inside the current suggestion post. | Replaces all existing tags with `In Progress...`, closes and locks the post, and renames it to `[IN PROGRESS] <old name>`. |
| `.exist` | Run directly inside the current suggestion post. | Replaces all existing tags with `Already exist`, closes and locks the post, renames it to `[ALREADY EXIST] <old name>`, and mentions the author. |
| `.reject <reason>` | Run directly inside the current suggestion post. | Replaces all existing tags with `Reject`, closes and locks the post, renames it to `[REJECTED] <old name>`, mentions the author, and sends the rejection reason. The reason is required and limited to 1,000 characters. |
| `.tba` | Run directly inside a suggestion post. | Removes all existing tags and keeps only `TBA`. It does not close, lock, or rename the post. |
| `.tbd` | Run directly inside a suggestion post. | Removes all existing tags and keeps only `TBD`. It does not close, lock, or rename the post. |
| `.accept`, `.maybe` | Run directly inside a managed post. | Applies the configured status tag, locks the post, and adds the corresponding status prefix. |

`.done`, `.in-progress`, `.exist`, `.reject`, `.tba`, and `.tbd` do not require a link. `.dupe` accepts either of the following forms:

```text
https://discord.com/channels/<guild_id>/<post_id>
https://discord.com/channels/<guild_id>/<post_id>/<message_id>
```

The bot validates the guild ID and refuses to process a `.dupe` target that is not in the configured `suggestion` Forum Channel.

### Prefix autocorrection

Set `prefix_autocorrect: true` to let the bot correct a close typo when there is exactly one unambiguous command within `prefix_max_distance`. For example, `.sloved` is not a registered alias, but it can be recognized as a typo for `.solved`. The bot reports the correction before applying the command. If multiple commands are equally close, it does nothing to avoid an unsafe guess.

## Configuration

Copy the sample configuration and fill in the bot token and guild ID:

```bash
cp config.example.yaml config.yaml
```

The sample already contains the configured `issues` and `suggestion` channel IDs and the tag names shown in the server setup. Never commit `config.yaml` or share the bot token.

```yaml
bot_token: "replace-with-discord-bot-token"
guild_id: "replace-with-server-id"
moderator_role_ids: []
replace_existing_tags: false
prefix_autocorrect: true
prefix_max_distance: 2
```

`replace_existing_tags: false` preserves tags that already exist in Discord and updates or appends declared tags. Set it to `true` only when the configured list should become the complete list of available tags for a channel.

## Create the Discord application

Create an Application and Bot User in the [Discord Developer Portal](https://discord.com/developers/applications). Invite the bot with the `bot` and `applications.commands` scopes.

The bot should be able to see both managed Forum Channels and should have at least the following permissions:

| Permission | Purpose |
| --- | --- |
| View Channel | Read Forum Channels and posts. |
| Send Messages in Threads | Send command results and moderation notices. |
| Manage Threads | Rename, archive, unarchive, lock, unlock, and manage posts. |
| Manage Channels | Update guidelines, available tags, and the `Require Tags when posting` flag. |

Enable **Message Content Intent** under the Bot settings. Without it, slash commands continue to work, but prefix commands cannot read message content. The bot must also be invited with the `applications.commands` scope so guild slash commands can be registered.

## Run locally

Go 1.22 or newer is required.

```bash
git clone https://github.com/sang765/discord-forum-bot.git
cd discord-forum-bot
cp config.example.yaml config.yaml
# edit config.yaml
make test
make vet
make build
./bin/discord-forum-bot
```

Use another configuration path with:

```bash
CONFIG_FILE=/path/to/config.yaml ./bin/discord-forum-bot
```

When the bot receives `Ready`, it overwrites the guild slash-command set and synchronizes all configured Forum Channels. After changing tags or guidelines, restart the bot or run `/forum-sync` for the relevant channel.

## Docker

```bash
docker build -t discord-forum-bot:latest .
docker run --rm \
  -e CONFIG_FILE=/app/config.yaml \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  discord-forum-bot:latest
```

## systemd

The repository includes `deploy/discord-forum-bot.service`. The following is an example installation on a Linux host:

```bash
sudo useradd --system --home /opt/discord-forum-bot --shell /usr/sbin/nologin discordbot
sudo mkdir -p /opt/discord-forum-bot
sudo chown -R discordbot:discordbot /opt/discord-forum-bot
sudo cp deploy/discord-forum-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now discord-forum-bot
sudo journalctl -u discord-forum-bot -f
```

## Deployment options

| Option | Trade-off | Cost | Setup complexity |
| --- | --- | --- | --- |
| Existing personal machine or Go-compatible server | No new infrastructure cost, but the machine must stay online and the operator manages restarts, updates, and security. | None if hardware already exists. | Low to medium. |
| Cloud server or Docker/Go-compatible platform | Independent 24/7 operation, but requires management of hosting cost, secrets, and operating-system updates. | Depends on the provider. | Medium. |

The binary, Dockerfile, and systemd unit are portable across both options.

## Testing

The project includes unit tests for configuration validation, tag merging, command parsing, link parsing, status restrictions, archived-thread pagination, and rejection reasons.

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./cmd/bot
```

Live testing requires a Discord token and a test server. Do not put secrets in source code or commit them to Git.

## References

[1]: https://docs.discord.com/developers/resources/channel "Discord Developer Documentation — Channels Resource"
[2]: https://docs.discord.com/developers/topics/threads "Discord Developer Documentation — Threads"
