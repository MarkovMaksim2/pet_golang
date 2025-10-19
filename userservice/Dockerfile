FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN apk add --no-cache make git build-base sqlite-dev

RUN go mod download

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
