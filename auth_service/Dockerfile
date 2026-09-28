FROM golang:1.27.1

COPY proto /src/proto
COPY pkg /src/pkg
COPY auth_service /src/auth_service
WORKDIR /src/auth_service

RUN go mod download
RUN go build -o auth ./cmd/auth/main.go

CMD ["./auth"]
