## services/update_grpc: refresh local contracts and module dependencies
.PHONY: services/update_grpc
services/update_grpc:
	$(MAKE) -C ../proto generate/rpc generate/openapi
	go mod tidy
