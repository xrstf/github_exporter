FROM golang:1.14-alpine as builder

WORKDIR /app/
COPY . .
RUN go build

FROM alpine:3.12

WORKDIR /app/
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/github_exporter .
EXPOSE 9612
ENTRYPOINT ["./github_exporter"]
