FROM golang:1.27.1-bookworm@sha256:8d48e12ec56735e9358640898b9d9b9fcca110612ed8a5567438c0a1baa24e66 AS development
WORKDIR /src
ENV GOTOOLCHAIN=local GOCACHE=/go-cache
COPY go.mod ./
COPY cmd ./cmd
CMD ["go", "test", "-count=1", "./..."]
