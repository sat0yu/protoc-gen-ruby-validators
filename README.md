# protoc-gen-ruby-validators

## Install
```sh
GO111MODULE=off go get github.com/sat0yu/protoc-gen-ruby-validators/cmd/protoc-gen-ruby-validators
```

## Usage
```sh
protoc --proto_path=. --proto_path=${GOPATH}/src --plugin=${GOPATH}/bin/protoc-gen-ruby-validators --ruby_out=. --ruby-validators_out=. *.proto
```
