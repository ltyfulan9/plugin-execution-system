FROM golang:1.22-bookworm AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN go test ./... && CGO_ENABLED=0 GOOS=linux go build -o /out/pes-server ./cmd/server && CGO_ENABLED=0 GOOS=linux go build -o /out/pesctl ./cmd/pesctl

FROM python:3.12-slim
WORKDIR /app
COPY --from=build /out/pes-server /usr/local/bin/pes-server
COPY --from=build /out/pesctl /usr/local/bin/pesctl
COPY configs ./configs
COPY plugins ./plugins
COPY web ./web
ENV SERVER_ADDR=:8080 STORAGE_DIR=/data PLUGIN_DIR=/app/plugins
VOLUME ["/data"]
EXPOSE 8080
ENTRYPOINT ["pes-server"]
