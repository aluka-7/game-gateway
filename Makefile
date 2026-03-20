STATUS:= `git status -s`

go:
	protoc --go_out=Mgoogle/protobuf/any.proto=github.com/gogo/protobuf/types,paths=source_relative:. .\proto\*.proto
	go test -v