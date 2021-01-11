FROM golang:1.15 AS builder
ENV CGO_ENABLED 0
ENV GOOS linux
ENV GOARCH amd64
ENV GOBIN /bin
WORKDIR /src
COPY go.mod go.sum /src/
RUN go mod download
COPY . /src
RUN go build -a -installsuffix nocgo -o /tmp/proxy ./cmd/proxy

FROM alpine
RUN adduser -D -u 1000 user
COPY --from=builder /tmp/proxy /usr/local/bin
EXPOSE 8080
USER 1000