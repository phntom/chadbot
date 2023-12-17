FROM golang:alpine AS builder
WORKDIR /app
COPY go.mod go.sum /app/
COPY internal /app/internal/
COPY cmd /app/cmd/
RUN go mod tidy
RUN go build -o chadbot /app/cmd/chadbot
ENV CGO_ENABLED 0
RUN go test ./...

FROM alpine:3.14
WORKDIR /app
RUN apk --no-cache add tzdata
COPY --from=builder /app/chadbot ./
ENV ADMIN_USERNAME=""
ENV AUTH_TOKEN=""
ENV BIND_IP="0.0.0.0"
ENV BIND_PORT="7357"
ENV CALLBACK_URL="http://chadbot.chat.svc:7357"
ENV CHAT_DOMAIN="https://kix.co.il"
ENV CHANNEL_PREFIX="chad2"
ENV CONFIG_CHANNEL="chadbot-config"
ENV EXTERNAL_MM_URL="https://kix.co.il"

EXPOSE 7357

CMD ["/app/chadbot"]
