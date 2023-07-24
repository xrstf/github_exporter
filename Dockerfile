FROM golang:1.20.6-alpine as builder

WORKDIR /app/
COPY . .
RUN go build

FROM alpine:3.17

RUN apk --no-cache add ca-certificates
COPY --from=builder /app/github_exporter .
EXPOSE 9612
ENTRYPOINT ["/github_exporter"]
