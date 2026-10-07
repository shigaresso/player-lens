FROM golang:1.27.1-bookworm@sha256:8d48e12ec56735e9358640898b9d9b9fcca110612ed8a5567438c0a1baa24e66 AS development
WORKDIR /src
ENV GOTOOLCHAIN=local GOCACHE=/go-cache
COPY go.mod ./
COPY cmd ./cmd
CMD ["sh", "-c", "go build -o /tmp/player-lens ./cmd/server && exec /tmp/player-lens"]

FROM development AS build
# 現在は標準ライブラリだけを使う。CGO 依存を導入するときはこの設定を見直す。
RUN --mount=type=cache,target=/go-cache CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/player-lens ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS production
COPY --from=build /out/player-lens /player-lens
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/player-lens"]
