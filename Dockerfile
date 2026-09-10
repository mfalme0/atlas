FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /atlas-server ./cmd/atlas-server/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /atlas-cli ./cmd/atlas-cli/

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /atlas-server /app/atlas-server
COPY --from=builder /atlas-cli /app/atlas-cli

EXPOSE 8080

ENTRYPOINT ["/app/atlas-server"]
