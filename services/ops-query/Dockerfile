FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /ops-query ./cmd/main.go

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
RUN addgroup -S app && adduser -S app -G app
USER app
WORKDIR /app
COPY --from=builder /ops-query .
EXPOSE 50063
ENTRYPOINT ["./ops-query"]
