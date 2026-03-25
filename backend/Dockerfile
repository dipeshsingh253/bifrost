FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/bifrost-backend .
RUN GOBIN=/out go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

FROM alpine:3.22

RUN addgroup -S bifrost && adduser -S bifrost -G bifrost

WORKDIR /app

COPY --from=builder /out/bifrost-backend /usr/local/bin/bifrost-backend
COPY --from=builder /out/migrate /usr/local/bin/migrate
COPY migrations ./migrations

USER bifrost

EXPOSE 8080

CMD ["bifrost-backend"]
