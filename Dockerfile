FROM golang:1.24-alpine AS builder
RUN apk add --no-cache build-base sqlite-dev
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o /out/license-server .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=builder /out/license-server /usr/local/bin/license-server
RUN mkdir -p /data
EXPOSE 2090
VOLUME ["/data"]
CMD ["license-server"]
