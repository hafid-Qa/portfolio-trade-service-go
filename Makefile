

buf:
	protoc -I proto/calc/v1 \
		--go_out=proto/gen --go_opt=paths=source_relative \
		--go-grpc_out=proto/gen --go-grpc_opt=paths=source_relative \
		calc.proto
