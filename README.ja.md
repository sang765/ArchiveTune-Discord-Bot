# ArchiveTune Bot

[English](README.md) · [日本語](README.ja.md) · [Tiếng Việt](README.vi.md)

**ArchiveTune Bot** は、Discord の Forum Channel におけるタグと投稿を管理するための **Go 製 Discord bot** です。bot はデータベースを使用せず、AI による本文処理も行いません。Discord を正規のデータソースとし、`config.yaml` に bot と channel の設定を保存します。

Discord では Forum の各投稿は thread として扱われます。利用可能な Forum タグは親 channel に保存され、個々の投稿に適用されたタグは thread に保存されます。[1] [2] そのため、この bot はリアルタイムイベントに Gateway を使用し、タグと投稿の決定的な操作に Discord REST API を使用します。

## 現在の対象範囲

| 分野 | 動作 |
| --- | --- |
| Forum 同期 | ガイドライン、設定済みタグ、`Require Tags when posting` フラグを同期します。 |
| suggestion の自動タグ | タグのない新規投稿に自動で `Maybe` を付けます。 |
| タグ管理 | 管理対象の投稿にタグを追加、削除、または置換します。 |
| 投稿管理 | command に応じて投稿名の変更、archive、unarchive、lock、unlock を行います。 |
| moderation workflow | issues と suggestion の status command、作成者への mention、拒否理由を提供します。 |
| デプロイ | Go binary、Docker、systemd に対応します。secret はローカル設定から読み込み、commit しません。 |
| Presence | `Why do I exist?` の custom status と Do Not Disturb (DND) status を表示します。 |

この bot は AI moderation、reaction 管理、履歴保存、本文に基づく自動分類を意図的に提供しません。対象範囲は Forum のタグと投稿を決定的に管理することです。

## 管理対象 channel

| Channel | ID | 特別な動作 |
| --- | --- | --- |
| `issues` | `1498327801923637439` | `.solved`、`.false`、`.false-report` はこの Forum Channel のみで利用できます。`.solved` または `.false` 成功時には投稿作成者を mention します。 |
| `suggestion` | `1498328044635422790` | タグのない新規投稿には `Maybe` を自動適用します。`/fix-suggestion`、`.dupe`、`.done`、`.in-progress`、`.exist`、`.reject`、`.tba`、`.tbd` がこの channel で動作します。 |

## Slash command

| Command | 使用方法 | 動作 |
| --- | --- | --- |
| `/help` | すべての channel で使用できます。 | すべての command を Embed で表示します。 |
| `/forum-sync channel:<Forum Channel>` | `config.yaml` を変更した後に実行します。 | ガイドライン、タグ、必須タグフラグを同期します。 |
| `/fix-suggestion` | moderator 用 command です。 | bot がアクセスできる active / archived の suggestion 投稿を走査し、タグのない投稿に `Maybe` を付けます。 |
| `/tag-add tag:<name>` | 投稿内で実行するか、`post_id` を指定します。 | 既存タグを削除せず、設定済みタグを追加します。 |
| `/tag-remove tag:<name>` | 投稿内で実行するか、`post_id` を指定します。 | 設定済みタグを削除します。 |
| `/post-rename name:<new name>` | 投稿内で実行するか、`post_id` を指定します。 | 管理対象の投稿名を変更します。 |
| `/post-state state:<open\|close\|lock\|unlock>` | 投稿内で実行するか、`post_id` を指定します。 | 投稿の archive または lock 状態を変更します。 |
| `/ytd url:<YouTube URL> type:<video\|audio\|thumbnail> [quality:<format id>]` | すべての member が使用できます。quality を省略すると interactive selector を開きます。 | yt-dlp で media をダウンロードし、temp.sh の一時リンクを返します。 |

## Prefix command

Prefix command は通常の message として送信します。利用には **Message Content Intent** と moderator 権限が必要です。ただし `.help` と `.ytd` はすべての member が使用できます。成功、エラー、usage、権限通知を含むすべての command response は、ArchiveTune Bot ブランドの Embed で送信されます。

| Command | 動作 |
| --- | --- |
| `.help` | すべての command を Embed で表示します。 |

### issues command

| Command | 動作 |
| --- | --- |
| `.solved` | `issues` 内で現在の workflow タグを `Solved` に置換し、投稿を lock して `[SOLVED] <旧タイトル>` に変更します。作成者を mention します。 |
| `.false` | `issues` 内で現在の workflow タグを `False report` に置換し、投稿を lock して `[FALSE REPORT] <旧タイトル>` に変更します。作成者を mention します。 |
| `.false-report` | `issues` における `False report` workflow の完全名 alias です。 |

### suggestion command

| Command | 使用方法 | 動作 |
| --- | --- | --- |
| `.dupe <post link または message link>` | Discord の投稿リンクまたはメッセージリンクが必要です。 | 既存タグをすべて `Duplicate` に置換し、対象 suggestion 投稿を close / lock し、`[DUPLICATE] <旧タイトル>` に変更して対象作成者を mention します。 |
| `.done` | 現在の suggestion 投稿内で直接実行します。 | 既存タグをすべて `Done` に置換し、投稿を close / lock して `[DONE] <旧タイトル>` に変更します。 |
| `.in-progress` | 現在の suggestion 投稿内で直接実行します。 | 既存タグをすべて `In Progress...` に置換し、投稿を close / lock して `[IN PROGRESS] <旧タイトル>` に変更します。 |
| `.exist` | 現在の suggestion 投稿内で直接実行します。 | 既存タグをすべて `Already exist` に置換し、投稿を close / lock して `[ALREADY EXIST] <旧タイトル>` に変更し、作成者を mention します。 |
| `.reject <理由>` | 現在の suggestion 投稿内で直接実行します。 | 既存タグをすべて `Reject` に置換し、投稿を close / lock して `[REJECTED] <旧タイトル>` に変更し、作成者を mention して理由を送信します。理由は必須で、1,000 文字以内です。 |
| `.tba` | suggestion 投稿内で直接実行します。 | 既存タグをすべて削除し、`TBA` のみを残します。close、lock、タイトル変更は行いません。 |
| `.tbd` | suggestion 投稿内で直接実行します。 | 既存タグをすべて削除し、`TBD` のみを残します。close、lock、タイトル変更は行いません。 |
| `.accept` | suggestion 投稿内で直接実行します。 | 既存タグをすべて `Accept` に置換し、投稿を close / lock して `[ACCEPTED] <旧タイトル>` に変更します。 |
| `.accepted` | suggestion 投稿内で直接実行します。 | `.accept` の alias です。 |
| `.maybe` | 管理対象の投稿内で直接実行します。 | 設定済み tag を適用し、投稿を lock して対応する status prefix を追加します。 |
| `.ytd <YouTube URL> type:<video\|audio\|thumbnail> [quality:<format id>]` | すべての member が使用できます。quality を省略すると format を先に表示します。 | yt-dlp で media をダウンロードし、temp.sh の一時リンクを返します。 |

`.done`、`.in-progress`、`.exist`、`.reject`、`.tba`、`.tbd` にリンクは必要ありません。`.dupe` は次の形式に対応します。

### YouTube downloader

Prefix command `.ytd` と slash command `/ytd` は YouTube と YouTube Music の URL に対応し、`video`、`audio`、`thumbnail` の三つの type を使用できます。

```text
.ytd https://youtu.be/dQw4w9WgXcQ?si=example type:video
.ytd https://youtu.be/dQw4w9WgXcQ?si=example type:audio quality:251
.ytd https://youtu.be/dQw4w9WgXcQ?si=example type:thumbnail
```

video または audio では、最初に `quality` を省略して interactive quality selector を開きます。dropdown から format を選び、**Download** を押すとダウンロードが始まります。format ID または `quality:best` を直接指定することもできます。ファイル名は sanitize 済みの title と type を使い、`My_Song_audio.opus`、`My_Video_video.mp4`、`My_Video_thumbnail.webp` のようになります。YouTube Music の audio は、artist metadata がある場合に `Song Title - Artist.opus` 形式となり、`_audio` suffix は付きません。yt-dlp が選択した元の extension は保持されます。ファイルは [temp.sh](https://temp.sh/) に upload され、リンクは一時的に約 3 日間有効です。Pterodactyl の startup script は、必要に応じて `.tools/media` に yt-dlp と ffmpeg を自動インストールします。無効にする場合は `AUTO_INSTALL_MEDIA_TOOLS=0` を設定してください。

```text
https://discord.com/channels/<guild_id>/<post_id>
https://discord.com/channels/<guild_id>/<post_id>/<message_id>
```

bot は guild ID を検証し、`.dupe` の対象が設定済みの `suggestion` Forum Channel に属していない場合は処理しません。

### Prefix の typo 自動修正

`prefix_autocorrect: true` にすると、`prefix_max_distance` の範囲内で、候補が一つだけの場合に近い typo を自動修正できます。例えば `.sloved` は登録済み alias ではありませんが、`.solved` の typo として認識できます。実行前に修正内容を message で通知します。同じ距離の候補が複数ある場合は、誤操作を避けるため何も実行しません。

## 設定

サンプル設定をコピーし、bot token と guild ID を入力します。

```bash
cp config.example.yaml config.yaml
```

サンプルには `issues` と `suggestion` の channel ID、および server setup に合わせたタグ名が含まれています。`config.yaml` を commit したり、bot token を共有したりしないでください。

```yaml
bot_token: "replace-with-discord-bot-token"
guild_id: "replace-with-server-id"
moderator_role_ids: []
replace_existing_tags: false
prefix_autocorrect: true
prefix_max_distance: 2
```

`replace_existing_tags: false` は Discord にすでに存在するタグを保持し、設定済みタグを更新または追加します。設定ファイルのリストを channel の完全なタグ一覧にしたい場合にのみ `true` にしてください。

### タグの emoji 設定

`emoji` フィールドは通常の Unicode emoji と Discord custom emoji の形式に対応します。静的な custom emoji は `<:name:id>`、アニメーションする custom emoji は `<a:name:id>` を使用します。

```yaml
- name: "ArchiveTune Version"
  emoji: "<:02V:1520005999040266240>"

- name: "Animated Status"
  emoji: "<a:loading:1520005999040266240>"
```

`emoji`、`emoji_id`、`emoji_animated` を分けて指定する形式も使用できます。custom emoji の形式を指定すると、bot が ID を抽出し、Discord Forum Tag の `emoji_id` だけを送信します。同じ tag payload に `emoji_name` と `emoji_id` の両方を設定すると Discord に拒否されるため、`emoji_name` は空のままにします。Unicode emoji は `emoji_name` で送信されます。静的 emoji に `emoji_animated: true` を設定しないでください。

## Discord Application の作成

[Discord Developer Portal](https://discord.com/developers/applications) で Application と Bot User を作成します。`bot` と `applications.commands` scope を付けて bot を server に invite してください。

bot は管理対象の二つの Forum Channel を閲覧でき、少なくとも次の権限を持つ必要があります。

| Permission | 目的 |
| --- | --- |
| View Channel | Forum Channel と投稿を読み取ります。 |
| Send Messages in Threads | command 結果と moderation 通知を送信します。 |
| Manage Threads | 投稿の名前変更、archive、unarchive、lock、unlock、管理を行います。 |
| Manage Channels | ガイドライン、available tags、`Require Tags when posting` を更新します。 |

Bot settings で **Message Content Intent** を有効にしてください。無効の場合、slash command は動作しますが prefix command は message 内容を読み取れません。また、guild slash command を登録するために `applications.commands` scope で invite する必要があります。

## ローカルで実行

Go 1.22 以降が必要です。

```bash
git clone https://github.com/sang765/discord-forum-bot.git
cd discord-forum-bot
cp config.example.yaml config.yaml
# config.yaml を編集
make test
make vet
make build
./bin/discord-forum-bot
```

別の設定ファイルを使う場合:

```bash
CONFIG_FILE=/path/to/config.yaml ./bin/discord-forum-bot
```

bot が `Ready` を受信すると、guild の slash-command セットを上書きし、設定済みの Forum Channel を同期します。タグまたはガイドラインを変更した後は、bot を再起動するか対象 channel で `/forum-sync` を実行してください。

## Docker

```bash
docker build -t discord-forum-bot:latest .
docker run --rm \
  -e CONFIG_FILE=/app/config.yaml \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  discord-forum-bot:latest
```

## Pterodactyl startup script

Repository には `run.sh` と `install-go.sh` が含まれています。Pterodactyl の startup command を `./${EXECUTABLE}`、`EXECUTABLE` を `run.sh`、`GO PACKAGE` を `./cmd/bot` に設定してください。script は project directory に移動し、container に Go があればそれを使用し、Go がなければ `install-go.sh` で user-local の Go 1.22.2 toolchain をダウンロードします。その後 Linux binary を build し、`CONFIG_FILE=./config.yaml` で bot を起動します。導入できない場合は既存の executable `./discord-forum-bot` を使用します。

```text
Startup Command: ./\${EXECUTABLE}
GO PACKAGE: ./cmd/bot
EXECUTABLE: run.sh
```

これにより、Pterodactyl File Manager で source を編集した後、server を restart するだけで自動的に再 build できます。初回起動には `go.dev` への HTTPS outbound access と、`.tools/go` に toolchain を保存するための空き容量が必要です。以降はダウンロード済みの toolchain を再利用します。container に network access がない場合は、事前に build した `discord-forum-bot` binary を upload してください。

source を自動更新するには、`.zip` archive を project root に upload して server を restart するだけです。`run.sh` は最新の ZIP を選択して検証・展開し、古い source を削除します。ただし `run.sh`、`config.yaml`、`.tools` は保持し、更新成功後に root の ZIP を削除します。壊れた archive や安全でない path を含む archive は拒否され、現在の source は削除されません。

`.tools/go` には local Go toolchain が保存されます。Go module download と build cache は `.tools/go-work` に保存され、`.tools/.discord-forum-bot-build-fingerprint` が source の状態を記録します。source に変更がなければ、restart 時に既存の binary を再利用し、再 build や dependency の再 download は行いません。

## systemd

Repository には `deploy/discord-forum-bot.service` が含まれています。Linux host でのインストール例:

```bash
sudo useradd --system --home /opt/discord-forum-bot --shell /usr/sbin/nologin discordbot
sudo mkdir -p /opt/discord-forum-bot
sudo chown -R discordbot:discordbot /opt/discord-forum-bot
sudo cp deploy/discord-forum-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now discord-forum-bot
sudo journalctl -u discord-forum-bot -f
```

## 運用方法

| 方法 | トレードオフ | 費用 | 設定の難易度 |
| --- | --- | --- | --- |
| 既存の個人 PC または Go 対応 server | 新しいインフラ費用は不要ですが、常時 online にし、restart、更新、security を管理する必要があります。 | 既存 hardware があれば不要です。 | 低〜中。 |
| Cloud server または Docker/Go 対応 platform | 24/7 で独立運用できますが、hosting 費用、secret、OS 更新を管理する必要があります。 | provider に依存します。 | 中。 |

Binary、Dockerfile、systemd unit は両方の方法で portable に利用できます。

## テスト

Project には config validation、tag merge、command parser、link parser、channel 制限、archived thread pagination、reject reason の unit test が含まれています。

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./cmd/bot
```

Live test には Discord token と test server が必要です。secret を source code に書いたり Git に commit したりしないでください。

## 参考資料

[1]: https://docs.discord.com/developers/resources/channel "Discord Developer Documentation — Channels Resource"
[2]: https://docs.discord.com/developers/topics/threads "Discord Developer Documentation — Threads"
