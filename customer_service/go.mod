module github.com/mikhaeris/sky-bank/customer_service

go 1.27.1

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.30.0
	github.com/ilyakaznacheev/cleanenv v1.5.0
	github.com/lib/pq v1.12.3
	github.com/mikhaeris/sky-bank/auth_service v0.0.0-20260916193320-5e8b1c6790ee
	github.com/mikhaeris/sky-bank/notification_service v0.0.0-20260916193320-5e8b1c6790ee
	google.golang.org/genproto/googleapis/api v0.0.0-20260911204522-f61a6ca850bd
	google.golang.org/grpc v1.83.2
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260904194346-d0f1323225a4 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	olympos.io/encoding/edn v0.0.0-20201019073823-d3554ca0b0a3 // indirect
)
