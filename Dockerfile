#--- Build stage
FROM golang:1.19 AS go-builder

WORKDIR /app/
COPY . .
RUN go build

#--- Image stage
FROM alpine:3.17.0

WORKDIR /app/
RUN apk --no-cache add ca-certificates=20220614-r4
COPY --from=go-builder /app/github-exporter .
EXPOSE 9612
ENTRYPOINT ["./github-exporter"]
