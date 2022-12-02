FROM golang:1.18.3 as builder

WORKDIR /go/src/github.com/vietanhduong/xcontroller

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o xcontroller ./cmd

FROM alpine:3.14

RUN apk --no-cache add ca-certificates git

COPY --from=builder /go/src/github.com/vietanhduong/xcontroller/xcontroller /usr/local/bin

ENTRYPOINT [ "/usr/local/bin/xcontroller" ]
