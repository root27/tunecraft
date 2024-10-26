FROM golang:1.23-alpine as builder

WORKDIR /go-app


COPY . .


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tunecraft .

FROM alpine


WORKDIR /app

RUN apk add --no-cache youtube-dl ffmpeg


COPY --from=builder /go-app/tunecraft .

ENTRYPOINT ["/app/tunecraft"]


