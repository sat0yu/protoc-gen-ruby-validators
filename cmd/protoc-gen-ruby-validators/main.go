package main

import (
    "log"
    "os"

    . "github.com/sat0yu/protoc-gen-ruby-validators/plugin"

    // download dependencies in ${GOPATH}/src
    _ "github.com/mwitkow/go-proto-validators"
)

func run() error {
    req, err := ParseRequest(os.Stdin)
    if err != nil {
        return err
    }

    resp := ProcessRequest(req)

    return EmitResponse(resp)
}

func main() {
    if err := run(); err != nil {
        log.Fatalln(err)
    }
}