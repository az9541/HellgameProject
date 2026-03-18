
Эта папка содержит сгенерированный код из `.proto` файлов.

# Установить компиляторы
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Запустить генерацию
❯ protoc --go_out=. --go_opt=module=HellgameProject --go-grpc_out=. --go-grpc_opt=module=HellgameProject proto/v1/engine.proto

❯ protoc --go_out=. --go-grpc_out=. proto/v1/engine.proto