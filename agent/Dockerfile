FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bifrost-agent .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates docker-cli
COPY --from=builder /out/bifrost-agent /usr/local/bin/bifrost-agent
ENTRYPOINT ["bifrost-agent"]
