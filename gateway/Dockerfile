FROM golang:1.27.1

COPY . /src
WORKDIR /src

RUN go mod download
RUN go build -o gateway ./cmd/gateway/

CMD ["./gateway"]
