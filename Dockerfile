FROM golang:1.17-alpine AS build
RUN apk add --update --no-cache git
WORKDIR /src
COPY ./go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /conoha_exporter

FROM alpine:3.14
WORKDIR /app

RUN apk add --update ca-certificates openssl && \
    update-ca-certificates && \
    rm -rf /var/cache/apk/*

EXPOSE 9330

COPY --from=build /conoha_exporter ./

ENTRYPOINT ./conoha_exporter
