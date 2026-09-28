.DEFAULT_GOAL := generate

PROTOC ?= protoc
PROTO_FLAGS := --proto_path=api --proto_path=third_party
RPC_PROTOS := $(sort $(wildcard api/*/v*/*.proto))

.PHONY: generate generate/rpc generate/openapi

generate: generate/rpc generate/openapi

generate/rpc:
	mkdir -p gen
	$(PROTOC) $(PROTO_FLAGS) \
		--go_out=paths=source_relative:gen \
		--go-grpc_out=paths=source_relative:gen \
		$(RPC_PROTOS)
	$(PROTOC) $(PROTO_FLAGS) \
		--grpc-gateway_out=paths=source_relative:gen \
		api/auth/v1/auth.proto \
		api/customer/v1/customer.proto

generate/openapi:
	mkdir -p openapi/auth/v1 openapi/customer/v1
	$(PROTOC) $(PROTO_FLAGS) \
		--openapi_out=paths=source_relative:openapi/auth/v1 \
		api/auth/v1/auth.proto
	$(PROTOC) $(PROTO_FLAGS) \
		--openapi_out=paths=source_relative:openapi/customer/v1 \
		api/customer/v1/customer.proto
