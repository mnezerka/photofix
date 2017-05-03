

Download dependencies
go get ./...

Build
go build photofix


Cross compiling

$ GOOS=windows GOARCH=386 go build -o bin/photofix_i386.exe photofix
$ GOOS=windows GOARCH=amd64 go build -o bin/photofix_amd64.exe photofix
