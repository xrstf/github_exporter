#--- Build stage
FROM golang:1.19.2 AS go-builder

WORKDIR /app/
COPY . .
RUN go build

#--- Image stage
FROM alpine:3.16.2

WORKDIR /app/
RUN apk --no-cache add ca-certificates=20220614-r0
COPY --from=go-builder /app/github-exporter .
EXPOSE 9612
ENTRYPOINT ["./github-exporter"]
