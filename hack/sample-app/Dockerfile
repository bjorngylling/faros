FROM golang:1.21 AS builder
WORKDIR /go/src
COPY go.mod go.sum .
RUN go mod download
COPY . .
ARG PACKAGE
RUN CGO_ENABLED=0 GOOS=linux go build -o app $PACKAGE

FROM scratch
COPY --from=builder /go/src/app .
CMD ["./app"]
