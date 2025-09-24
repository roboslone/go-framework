#!/usr/bin/zsh

go get
golangci-lint run --no-config .
go test ./...
