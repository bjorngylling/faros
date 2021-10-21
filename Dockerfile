FROM golang:1.17 AS builder
WORKDIR /go/src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

FROM scratch
COPY --from=builder /go/src/app .
CMD ["./app"]