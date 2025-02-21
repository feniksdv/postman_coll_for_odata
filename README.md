Запустить проект: make run

windows
GOOS=windows GOARCH=amd64 go build -o sherp_odata_postman_collection.exe ./cmd


macos
go build -o sherp_odata_postman_collection_macos ./cmd

linux
GOOS=linux GOARCH=amd64 go build -o sherp_odata_postman_collection_linux ./cmd

