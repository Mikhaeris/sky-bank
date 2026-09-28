FROM golang:1.27.1

COPY proto /src/proto
COPY pkg /src/pkg
COPY gateway /src/gateway
WORKDIR /src/gateway

RUN go mod download
RUN go build -o gateway ./cmd/gateway/

CMD ["./gateway"]
