FROM golang:alpine AS builder
RUN apk add --no-cache git make
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o zendesk-exporter ./cmd/

FROM alpine
WORKDIR /app
COPY --from=builder /app/zendesk-exporter .
EXPOSE 9101
ENTRYPOINT ["/app/zendesk-exporter"]
