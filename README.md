# Discord Forum Tag Manager Bot

Bot Discord viết bằng **Go** với một nhiệm vụ duy nhất: quản lý tag và post trong các **Forum Channel** của server. Project không dùng database; cấu hình nằm trong `config.yaml`, còn Discord là nguồn dữ liệu chính.

Discord biểu diễn mỗi post trong Forum Channel như một thread, và các tag khả dụng nằm ở `available_tags`; tag đang áp dụng trên post nằm ở `applied_tags`.[1] Vì vậy bot sử dụng Gateway để nhận lệnh và REST API để đồng bộ channel, chỉnh tag, đổi tên, đóng/mở hoặc khóa/mở khóa post.[1] [2]

## Phạm vi hiện tại

| Nhóm | Chức năng |
| --- | --- |
| Đồng bộ channel | Cập nhật Post Guidelines, tạo/cập nhật tag theo cấu hình và bật/tắt `Require Tags when posting`. |
| Quản lý tag | Thêm hoặc gỡ tag trên post hiện tại hoặc post được chỉ định bằng ID. |
| Quản lý post | Đổi tên, mở, đóng, khóa và mở khóa post. |
| Phân quyền | Cho phép người dùng có `Manage Threads`, `Administrator` hoặc role ID được khai báo trong cấu hình. |
| Triển khai | Có thể chạy bằng binary, Docker hoặc systemd; token chỉ đọc từ file cấu hình không commit. |

Bot cố ý **không** xử lý nội dung AI, không lưu lịch sử, không quản lý reaction và không tự động đổi tag dựa trên nội dung. Đây là ranh giới để project chỉ làm một nhiệm vụ ổn định: quản lý tag và post.

## Các lệnh slash

| Lệnh | Cách dùng | Ý nghĩa |
| --- | --- | --- |
| `/forum-sync channel:<Forum Channel>` | Chạy sau khi sửa `config.yaml` | Đồng bộ guideline, tag và cờ bắt buộc tag. |
| `/tag-add tag:<tên>` | Chạy trong post; hoặc thêm `post_id` | Thêm tag đã khai báo cho Forum Channel. |
| `/tag-remove tag:<tên>` | Chạy trong post; hoặc thêm `post_id` | Gỡ tag khỏi post. |
| `/post-rename name:<tên mới>` | Chạy trong post; hoặc thêm `post_id` | Đổi tên post. |
| `/post-state state:<open\|close\|lock\|unlock>` | Chạy trong post; hoặc thêm `post_id` | Thay đổi trạng thái archive hoặc lock của post. |
| `.solved` | Gửi trực tiếp trong post trong `issues` | Gắn tag `Solved`, khóa post và đổi tên thành `[SOLVED] ...`. Chỉ hoạt động trong issues channel `1498327801923637439`. |
| `.dupe <post link hoặc message link>` | Gửi trong bất kỳ channel nào có thể dùng command | Chỉ xử lý post thuộc `suggestion`: xóa tag cũ, gắn `Duplicate`, đóng, khóa và đổi tên thành `[DUPLICATE] ...`, sau đó mention tác giả post. |
| `.done` | Gửi trực tiếp trong post suggestion | Thay toàn bộ tag bằng `Done`, đóng, khóa và đổi tên thành `[DONE] ...`. |
| `.in-progress` | Gửi trực tiếp trong post suggestion | Thay toàn bộ tag bằng `In Progress...`, đóng, khóa và đổi tên thành `[IN PROGRESS] ...`. |
| `.exist` | Gửi trực tiếp trong post suggestion | Thay toàn bộ tag bằng `Already exist`, đóng, khóa, đổi tên thành `[ALREADY EXIST] ...` và mention tác giả post. |
| `/fix-suggestion` | Slash command dành cho moderator | Quét active và archived post trong suggestion channel `1498328044635422790`, sau đó gắn `Maybe` cho post chưa có tag. |
| `.accept`, `.reject`, `.maybe` | Gửi trực tiếp trong post | Áp dụng tag tương ứng, khóa post và thêm prefix trạng thái vào tên. |
| `.duplicate`, `.already-exist`, `.tba`, `.tbd` | Gửi trực tiếp trong post | Áp dụng tag tương ứng, khóa post và thêm prefix trạng thái vào tên. |
| `.problem`, `.question`, `.stable`, `.nightly`, `.false`, `.false-report`, `.meta` | Gửi trực tiếp trong post | Áp dụng tag tương ứng, khóa post và thêm prefix trạng thái vào tên. `.false` gắn tag `False report` và đổi tên thành `[FALSE REPORT] ...`; `.false` và `.false-report` chỉ chạy trong issues channel `1498327801923637439`. |

Nếu lệnh tag được chạy trước khi bot đồng bộ channel, bot sẽ báo rằng tag chưa có Discord ID và yêu cầu chạy `/forum-sync` trước. Điều này tránh việc áp dụng một ID không tồn tại.

## Cấu hình

Sao chép file mẫu rồi điền token, guild ID và channel ID:

```bash
cp config.example.yaml config.yaml
```

Trong `config.yaml`, mỗi phần tử `channels` tương ứng với một Forum Channel. Hai cấu hình mẫu đã bám theo các ảnh bạn gửi: `issues` có các tag `Problem`, `Question`, `Stable Version`, `Nightly Version`, `False report`, `Solved`, `meta`; `suggestion` có các tag `Accept`, `Reject`, `Done`, `In Progress...`, `Maybe`, `Duplicate`, `Already exist`, `TBA`, `TBD`.

`replace_existing_tags: false` là lựa chọn an toàn: bot giữ lại các tag đang có trong Discord rồi cập nhật hoặc bổ sung tag được khai báo. Nếu đặt thành `true`, bot sẽ gửi danh sách tag được khai báo làm danh sách chính của channel; chỉ nên dùng sau khi kiểm tra kỹ vì các tag ngoài cấu hình sẽ bị loại khỏi channel.

## Tạo Discord Application

Trong Discord Developer Portal, tạo Application và thêm Bot User. Khi mời bot vào server, bật các scope `bot` và `applications.commands`. Bot cần nhìn thấy các Forum Channel được quản lý và có tối thiểu các quyền sau:

| Quyền | Mục đích |
| --- | --- |
| View Channel | Đọc Forum Channel và post. |
| Send Messages in Threads | Gửi phản hồi trong thread nếu sau này cần mở rộng. |
| Manage Threads | Đổi tên, archive/unarchive, lock/unlock và quản lý post. |
| Manage Channels | Cập nhật guidelines, available tags và cờ `Require Tags when posting`. |
|
Discord ghi nhận rằng Forum Channel chỉ chứa thread/post và không nhận message trực tiếp; việc tạo hoặc quản lý post dùng các endpoint thread tương ứng.[2] Các thread bị khóa hạn chế việc chỉnh sửa nếu không có `Manage Threads`, vì vậy nên cấp quyền này cho bot.[2]

## Chạy local

Cần Go 1.22 trở lên:

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

Có thể đổi vị trí cấu hình bằng biến môi trường:

```bash
CONFIG_FILE=/path/to/config.yaml ./bin/discord-forum-bot
```

Sau khi bot online, handler `Ready` sẽ đăng ký lại bộ slash command theo guild, đồng bộ toàn bộ channel trong cấu hình một lần và lắng nghe prefix command trong message. Khi sửa tag hoặc guidelines, chạy lại bot hoặc dùng `/forum-sync` cho từng channel.

Để prefix command hoạt động, cần bật **Message Content Intent** cho Bot User trong Discord Developer Portal. Nếu không bật intent này, slash command vẫn hoạt động nhưng bot sẽ không đọc được nội dung bắt đầu bằng dấu chấm.

Có thể bật `prefix_autocorrect: true` để bot tự sửa một typo gần đúng nếu chỉ có một lệnh phù hợp trong khoảng cách `prefix_max_distance`. Ví dụ, `.sloved` không còn là lệnh hợp lệ và không được đăng ký như alias, nhưng sẽ được nhận diện là lỗi chính tả của `.solved`, sau đó bot báo rõ lệnh đã được sửa trước khi thực hiện. Nếu có nhiều lệnh cùng gần như nhau, bot sẽ không tự đoán để tránh thao tác nhầm.

Lệnh `.dupe` chấp nhận cả post link dạng `https://discord.com/channels/<guild_id>/<post_id>` và message link dạng `https://discord.com/channels/<guild_id>/<post_id>/<message_id>`. Bot kiểm tra guild ID, xác định post từ channel ID trong link và từ chối nếu post không thuộc Forum Channel `suggestion`.

Forum Channel `issues` có ID `1498327801923637439`; các lệnh `.solved` và `.false` được giới hạn vào channel này. Khi `.solved` hoặc `.false` thành công, bot sẽ gửi một message mention tác giả của post. Forum Channel `suggestion` có ID `1498328044635422790`. Với mọi post mới chưa có tag, bot sẽ tự động gắn `Maybe` ngay khi nhận được event tạo thread. `/fix-suggestion` dùng để sửa các post cũ chưa có tag, bao gồm cả post active và archived mà bot có quyền truy cập.

## Chạy bằng Docker

```bash
docker build -t discord-forum-bot:latest .
docker run --rm \
  -e CONFIG_FILE=/app/config.yaml \
  -v "$PWD/config.yaml:/app/config.yaml:ro" \
  discord-forum-bot:latest
```

## Chạy bằng systemd

File `deploy/discord-forum-bot.service` là unit mẫu. Có thể đặt project tại `/opt/discord-forum-bot`, tạo user hệ thống `discordbot`, rồi build binary vào `bin/`:

```bash
sudo useradd --system --home /opt/discord-forum-bot --shell /usr/sbin/nologin discordbot
sudo mkdir -p /opt/discord-forum-bot
sudo chown -R discordbot:discordbot /opt/discord-forum-bot
sudo cp deploy/discord-forum-bot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now discord-forum-bot
sudo journalctl -u discord-forum-bot -f
```

## Hai cách vận hành

Project được đóng gói portable để server có thể chọn cách vận hành mà không phải đổi mã nguồn:

| Cách vận hành | Trade-off | Chi phí | Độ phức tạp thiết lập |
| --- | --- | --- | --- |
| Chạy trên máy cá nhân hoặc máy chủ Go-compatible hiện có | Không tốn hạ tầng mới, nhưng máy phải luôn online và người vận hành tự lo restart, cập nhật và bảo mật. | Không phát sinh nếu đã có máy. | Thấp đến trung bình. |
| Chạy trên một máy chủ cloud hoặc nền tảng hỗ trợ Docker/Go | Bot độc lập với máy cá nhân và có thể chạy 24/7; đổi lại cần quản lý chi phí, secret và cập nhật hệ điều hành. | Phụ thuộc nhà cung cấp. | Trung bình. |

Mình chưa tự chọn nơi deploy vì yêu cầu hiện tại mới dừng ở việc tạo project. Cả hai cách đều dùng cùng binary, `Dockerfile` và `systemd` unit trong repository.

## Kiểm thử

Project hiện có unit test cho validation cấu hình và việc merge tag, đồng thời chạy được:

```bash
GOTOOLCHAIN=local go test ./...
GOTOOLCHAIN=local go vet ./...
```

Kiểm thử live cần token Discord và một server test; không nên đưa token vào source code hoặc commit vào Git.

## Tài liệu tham chiếu

[1]: https://docs.discord.com/developers/resources/channel "Discord Developer Documentation — Channels Resource"
[2]: https://docs.discord.com/developers/topics/threads "Discord Developer Documentation — Threads"
