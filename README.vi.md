# ArchiveTune Bot

[English](README.md) · [日本語](README.ja.md) · [Tiếng Việt](README.vi.md)

**ArchiveTune Bot** là Discord bot viết bằng **Go**, tập trung vào một nhiệm vụ: quản lý tag và post trong Discord Forum Channel. Bot không sử dụng database và không xử lý nội dung bằng AI. Discord là nguồn dữ liệu chính, còn `config.yaml` lưu cấu hình bot và channel.

Discord biểu diễn mỗi post trong Forum Channel như một thread. Tag khả dụng được lưu ở channel cha, còn tag áp dụng cho từng post được lưu trên thread.[1] [2] Vì vậy bot dùng Gateway cho event thời gian thực và Discord REST API cho các thao tác tag và post có quy tắc rõ ràng.

## Phạm vi hiện tại

| Nhóm | Chức năng |
| --- | --- |
| Đồng bộ Forum | Đồng bộ guideline, tag đã khai báo và cờ `Require Tags when posting`. |
| Tự động gắn tag suggestion | Tự động gắn `Maybe` cho mọi post mới trong suggestion channel nếu post chưa có tag. |
| Quản lý tag | Thêm, gỡ hoặc thay toàn bộ tag trên post được quản lý. |
| Quản lý post | Đổi tên, archive, unarchive, lock và unlock post theo từng command. |
| Quy trình moderation | Có status command cho issues và suggestion, mention tác giả và gửi lý do từ chối. |
| Triển khai | Hỗ trợ binary Go, Docker hoặc systemd. Secret chỉ đọc từ file cấu hình local và không commit. |
| Presence | Hiển thị custom status `Why do I exist?` cùng trạng thái Do Not Disturb (DND). |

Bot cố ý không cung cấp AI moderation, quản lý reaction, lưu lịch sử hoặc tự phân loại dựa trên nội dung. Ranh giới của project là quản lý tag và post trong Forum một cách deterministic.

## Các channel được quản lý

| Channel | ID | Hành vi đặc biệt |
| --- | --- | --- |
| `issues` | `1498327801923637439` | `.solved`, `.false` và `.false-report` chỉ được dùng tại đây. Khi `.solved` hoặc `.false` thành công, bot mention tác giả post. |
| `suggestion` | `1498328044635422790` | Post mới chưa có tag tự động nhận `Maybe`. `/fix-suggestion`, `.dupe`, `.done`, `.in-progress`, `.exist`, `.reject`, `.tba` và `.tbd` hoạt động tại đây. |

## Slash command

| Command | Cách dùng | Hành vi |
| --- | --- | --- |
| `/help` | Có thể dùng ở mọi channel. | Hiển thị toàn bộ danh sách command trong Embed. |
| `/forum-sync channel:<Forum Channel>` | Chạy sau khi sửa `config.yaml`. | Đồng bộ guideline, tag và cờ bắt buộc tag. |
| `/fix-suggestion` | Dành cho moderator. | Quét các suggestion post active và archived mà bot có quyền truy cập, sau đó gắn `Maybe` cho post chưa có tag. |
| `/tag-add tag:<name>` | Chạy trong post hoặc truyền `post_id`. | Thêm tag đã cấu hình mà không xóa tag hiện tại. |
| `/tag-remove tag:<name>` | Chạy trong post hoặc truyền `post_id`. | Gỡ tag đã cấu hình. |
| `/post-rename name:<new name>` | Chạy trong post hoặc truyền `post_id`. | Đổi tên post được quản lý. |
| `/post-state state:<open\|close\|lock\|unlock>` | Chạy trong post hoặc truyền `post_id`. | Thay đổi trạng thái archive hoặc lock của post. |

## Prefix command

Prefix command được gửi như message thông thường. Bot cần **Message Content Intent** và người dùng cần quyền moderator, ngoại trừ `.help` có thể dùng bởi mọi thành viên. Tất cả phản hồi command, gồm thành công, lỗi, usage và thông báo quyền, đều được gửi dưới dạng Embed thương hiệu ArchiveTune Bot.

| Command | Hành vi |
| --- | --- |
| `.help` | Hiển thị toàn bộ danh sách command trong Embed. |

### Command cho issues

| Command | Hành vi |
| --- | --- |
| `.solved` | Trong `issues`, thay tag workflow hiện tại bằng `Solved`, lock post và đổi tên thành `[SOLVED] <tên cũ>`. Bot mention tác giả. |
| `.false` | Trong `issues`, thay tag workflow hiện tại bằng `False report`, lock post và đổi tên thành `[FALSE REPORT] <tên cũ>`. Bot mention tác giả. |
| `.false-report` | Alias đầy đủ của workflow `False report` trong `issues`. |

### Command cho suggestion

| Command | Cách dùng | Hành vi |
| --- | --- | --- |
| `.dupe <post link hoặc message link>` | Bắt buộc có Discord post/message link. | Thay toàn bộ tag bằng `Duplicate`, close và lock suggestion post đích, đổi tên thành `[DUPLICATE] <tên cũ>` và mention tác giả đích. |
| `.done` | Gửi trực tiếp trong suggestion post hiện tại. | Thay toàn bộ tag bằng `Done`, close và lock post, đổi tên thành `[DONE] <tên cũ>`. |
| `.in-progress` | Gửi trực tiếp trong suggestion post hiện tại. | Thay toàn bộ tag bằng `In Progress...`, close và lock post, đổi tên thành `[IN PROGRESS] <tên cũ>`. |
| `.exist` | Gửi trực tiếp trong suggestion post hiện tại. | Thay toàn bộ tag bằng `Already exist`, close và lock post, đổi tên thành `[ALREADY EXIST] <tên cũ>` và mention tác giả. |
| `.reject <lý do>` | Gửi trực tiếp trong suggestion post hiện tại. | Thay toàn bộ tag bằng `Reject`, close và lock post, đổi tên thành `[REJECTED] <tên cũ>`, mention tác giả và gửi lý do. Lý do bắt buộc, tối đa 1.000 ký tự. |
| `.tba` | Gửi trực tiếp trong suggestion post. | Xóa toàn bộ tag cũ và chỉ giữ `TBA`. Không close, lock hoặc đổi tên post. |
| `.tbd` | Gửi trực tiếp trong suggestion post. | Xóa toàn bộ tag cũ và chỉ giữ `TBD`. Không close, lock hoặc đổi tên post. |
| `.accept` | Gửi trực tiếp trong suggestion post. | Thay toàn bộ tag bằng `Accept`, close và lock post, đổi tên thành `[ACCEPTED] <tên cũ>`. |
| `.accepted` | Gửi trực tiếp trong suggestion post. | Alias của `.accept`. |
| `.maybe` | Gửi trực tiếp trong post được quản lý. | Áp dụng tag đã cấu hình, lock post và thêm prefix trạng thái tương ứng. |

`.done`, `.in-progress`, `.exist`, `.reject`, `.tba` và `.tbd` không cần link. Riêng `.dupe` hỗ trợ hai dạng link:

```text
https://discord.com/channels/<guild_id>/<post_id>
https://discord.com/channels/<guild_id>/<post_id>/<message_id>
```

Bot kiểm tra guild ID và từ chối xử lý nếu target của `.dupe` không nằm trong Forum Channel `suggestion` đã cấu hình.

### Tự sửa lỗi chính tả prefix

Đặt `prefix_autocorrect: true` để bot tự sửa typo gần đúng khi chỉ có một command duy nhất nằm trong khoảng `prefix_max_distance`. Ví dụ `.sloved` không phải alias đã đăng ký, nhưng có thể được nhận diện là typo của `.solved`. Bot sẽ báo command đã được sửa trước khi thực thi. Nếu có nhiều command có độ gần bằng nhau, bot không tự đoán để tránh thao tác nhầm.

## Cấu hình

Sao chép config mẫu rồi điền token bot và guild ID:

```bash
cp config.example.yaml config.yaml
```

Config mẫu đã có sẵn ID của hai channel `issues` và `suggestion` cùng danh sách tag theo cấu hình server. Không commit `config.yaml` và không chia sẻ bot token.

```yaml
bot_token: "replace-with-discord-bot-token"
guild_id: "replace-with-server-id"
moderator_role_ids: []
replace_existing_tags: false
prefix_autocorrect: true
prefix_max_distance: 2
```

`replace_existing_tags: false` giữ lại các tag đã tồn tại trong Discord và cập nhật hoặc bổ sung các tag được khai báo. Chỉ đặt `true` khi danh sách trong config phải trở thành toàn bộ danh sách tag khả dụng của channel.

## Tạo Discord Application

Tạo Application và Bot User trong [Discord Developer Portal](https://discord.com/developers/applications). Invite bot với scope `bot` và `applications.commands`.

Bot cần nhìn thấy hai Forum Channel và tối thiểu có các quyền sau:

| Permission | Mục đích |
| --- | --- |
| View Channel | Đọc Forum Channel và post. |
| Send Messages in Threads | Gửi phản hồi command và thông báo moderation. |
| Manage Threads | Đổi tên, archive, unarchive, lock, unlock và quản lý post. |
| Manage Channels | Cập nhật guideline, available tags và cờ `Require Tags when posting`. |

Bật **Message Content Intent** trong phần Bot settings. Nếu không bật, slash command vẫn hoạt động nhưng prefix command không đọc được nội dung message. Bot cũng phải được invite với scope `applications.commands` để đăng ký slash command theo guild.

## Chạy local

Cần Go 1.22 trở lên.

```bash
git clone https://github.com/sang765/discord-forum-bot.git
cd discord-forum-bot
cp config.example.yaml config.yaml
# sửa config.yaml
make test
make vet
make build
./bin/discord-forum-bot
```

Dùng config path khác bằng:

```bash
CONFIG_FILE=/path/to/config.yaml ./bin/discord-forum-bot
```

Khi nhận event `Ready`, bot sẽ overwrite bộ slash command của guild và đồng bộ mọi Forum Channel đã cấu hình. Sau khi sửa tag hoặc guideline, restart bot hoặc chạy `/forum-sync` cho channel tương ứng.

## Docker

```bash
docker build -t discord-forum-bot:latest .
docker run --rm \
  -e CONFIG_FILE=/app/config.yaml \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  discord-forum-bot:latest
```

## Startup script cho Pterodactyl

Repository có file `run.sh` và `install-go.sh`. Đặt lệnh khởi động Pterodactyl là `./${EXECUTABLE}`, đặt `EXECUTABLE` thành `run.sh` và `GO_PACKAGE` thành `./cmd/bot`. Script sẽ chuyển vào thư mục project, dùng Go có sẵn trong container hoặc tự tải Go 1.22.2 vào thư mục local thông qua `install-go.sh` nếu image không có Go. Sau đó script build binary Linux mới và chạy bot với `CONFIG_FILE=./config.yaml`. Nếu cài đặt không được, script sẽ dùng binary executable có sẵn là `./discord-forum-bot`.

```text
Lệnh khởi động: ./\${EXECUTABLE}
GO PACKAGE: ./cmd/bot
EXECUTABLE: run.sh
```

Nhờ vậy, mỗi lần bạn sửa source bằng File Manager rồi restart server, bot sẽ tự build lại mà không cần chạy `go build` thủ công. Lần chạy đầu cần HTTPS outbound tới `go.dev` và đủ dung lượng để lưu Go local trong `.tools/go`; các lần sau sẽ dùng lại toolchain đã tải. Nếu container không có mạng, hãy upload binary đã build với tên `discord-forum-bot`.

Để cập nhật source tự động, chỉ cần upload một file `.zip` vào thư mục root rồi restart server. `run.sh` sẽ chọn ZIP mới nhất, kiểm tra và giải nén, xóa source cũ, giữ lại `run.sh`, `config.yaml` và `.tools`, sau đó xóa các file ZIP ở root khi thay thế thành công. Nếu archive bị lỗi hoặc chứa đường dẫn không an toàn, script sẽ từ chối và giữ nguyên source hiện tại.

Thư mục `.tools/go` chứa Go toolchain local. Các module Go và build cache được lưu trong `.tools/go-work`, còn `.tools/.discord-forum-bot-build-fingerprint` ghi nhận trạng thái source. Khi source không thay đổi, các lần restart sau sẽ dùng lại binary hiện tại và không build hoặc tải dependency lại.

## systemd

Repository có file `deploy/discord-forum-bot.service`. Ví dụ cài đặt trên Linux host:

```bash
sudo useradd --system --home /opt/discord-forum-bot --shell /usr/sbin/nologin discordbot
sudo mkdir -p /opt/discord-forum-bot
sudo chown -R discordbot:discordbot /opt/discord-forum-bot
sudo cp deploy/discord-forum-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now discord-forum-bot
sudo journalctl -u discord-forum-bot -f
```

## Phương án vận hành

| Phương án | Đánh đổi | Chi phí | Độ phức tạp |
| --- | --- | --- | --- |
| Máy cá nhân hoặc server Go-compatible hiện có | Không phát sinh hạ tầng mới, nhưng máy phải online và người vận hành tự quản lý restart, update và bảo mật. | Không phát sinh nếu đã có phần cứng. | Thấp đến trung bình. |
| Cloud server hoặc nền tảng hỗ trợ Docker/Go | Chạy độc lập 24/7, nhưng cần quản lý chi phí, secret và cập nhật hệ điều hành. | Tùy nhà cung cấp. | Trung bình. |

Binary, Dockerfile và systemd unit đều portable giữa hai phương án.

## Kiểm thử

Project có unit test cho validation config, merge tag, parser command, parser link, giới hạn channel, pagination archived thread và lý do reject.

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./cmd/bot
```

Kiểm thử live cần Discord token và server test. Không đặt secret trong source code hoặc commit vào Git.

## Tài liệu tham chiếu

[1]: https://docs.discord.com/developers/resources/channel "Discord Developer Documentation — Channels Resource"
[2]: https://docs.discord.com/developers/topics/threads "Discord Developer Documentation — Threads"
