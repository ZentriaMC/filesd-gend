FROM golang:alpine AS builder

# Build app
WORKDIR $GOPATH/src/github.com/ZentriaMC/filesd-gend
COPY . .
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-w -s" -o /filesd-gend ./cmd/filesd-gend

FROM scratch
COPY --from=builder /filesd-gend /filesd-gend

CMD ["/filesd-gend"]
