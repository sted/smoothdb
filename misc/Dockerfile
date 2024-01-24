## first stage
FROM golang:latest AS builder
COPY . .
ENV GO111MODULE=on
RUN go mod download
RUN GOOS=linux CGO_ENABLED=0 go build -o smoothdb

## second stage
FROM alpine
WORKDIR /app
COPY --from=builder . .
EXPOSE 4000
CMD /app/start.sh