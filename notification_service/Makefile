## grpc/generate: genereate proto contract
.PHONY: grpc/generate
grpc/generate:
	mkdir -p ./pkg
	protoc \
		--proto_path=./api \
		--proto_path=./third_party \
		--go_out=paths=source_relative:./pkg \
		--go-grpc_out=paths=source_relative:./pkg \
		--grpc-gateway_out=paths=source_relative:./pkg \
		notification/v1/notification.proto
