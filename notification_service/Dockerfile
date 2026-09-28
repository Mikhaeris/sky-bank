FROM golang:1.27.1

COPY proto /src/proto
COPY pkg /src/pkg
COPY notification_service /src/notification_service
WORKDIR /src/notification_service

RUN go mod download
RUN go build -o notification ./cmd/notification/main.go

CMD ["./notification"]
