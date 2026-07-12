FROM golang:1.19 AS builder
ARG HTTP_PROXY
ARG HTTPS_PROXY
ARG ALL_PROXY
ARG http_proxy
ARG https_proxy
ARG all_proxy
ENV HTTP_PROXY=${HTTP_PROXY}
ENV HTTPS_PROXY=${HTTPS_PROXY}
ENV ALL_PROXY=${ALL_PROXY}
ENV http_proxy=${http_proxy}
ENV https_proxy=${https_proxy}
ENV all_proxy=${all_proxy}
WORKDIR /src
COPY . .
RUN go mod tidy && go build -ldflags "-s -w" -o /out/forum-watch-bot ./cmd/forum-watch-bot

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/forum-watch-bot /app/forum-watch-bot
COPY config.example.json /app/config.example.json
CMD ["/app/forum-watch-bot", "/app/config.json"]
