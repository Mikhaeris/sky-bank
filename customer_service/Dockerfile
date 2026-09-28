FROM golang:1.27.1

COPY proto /src/proto
COPY pkg /src/pkg
COPY customer_service /src/customer_service
WORKDIR /src/customer_service

RUN go mod download
RUN go build -o customer ./cmd/customer/main.go

CMD ["./customer"]
