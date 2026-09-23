FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod ./
# If there's no go.sum, that's fine.
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /agent ./cmd/agent

FROM alpine:3.20
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /agent .
CMD ["./agent"]
