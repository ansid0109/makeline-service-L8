FROM golang:1.22.5-alpine AS builder

WORKDIR /app
ARG APP_VERSION=0.1.0
COPY . /app

RUN go build -ldflags "-X main.version=$APP_VERSION" -o main .


FROM alpine:latest AS runner

ARG APP_VERSION=0.1.0
COPY --from=builder /app/main .
EXPOSE 3001
ENV APP_VERSION=$APP_VERSION

CMD ["./main"]