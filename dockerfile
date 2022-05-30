## Build
FROM golang:1.18.1-alpine AS buildenv

WORKDIR /app

ADD . .

ENV GO111MODULE=on

RUN go mod download

RUN GOOS=linux GOARCH=arm64 go build -buildvcs=false -o /green-ds

## Deploy
FROM alpine

WORKDIR /

COPY --from=buildenv  /green-ds /green-ds

EXPOSE 8081

CMD ["/green-ds"]