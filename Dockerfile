FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /wa-server ./cmd/server

FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /wa-server .

EXPOSE 9090

ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=9090
ENV DB_HOST=postgres
ENV DB_PORT=5432
ENV RABBITMQ_HOST=rabbitmq
ENV RABBITMQ_PORT=5672
ENV REDIS_HOST=redis
ENV REDIS_PORT=6379

ENTRYPOINT ["/app/wa-server"]