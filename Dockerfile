FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /out/discord-forum-bot ./cmd/bot

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/discord-forum-bot /app/discord-forum-bot
COPY config.example.yaml /app/config.example.yaml
USER nonroot:nonroot
ENTRYPOINT ["/app/discord-forum-bot"]
