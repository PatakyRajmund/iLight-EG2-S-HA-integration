package main

import (
	"host/lamp/jsonrpcserver"
	"sync"
)

var wg sync.WaitGroup

func main() {
	jsonrpcserver.Run(&wg)
}
