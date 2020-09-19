package main

import (
	"fmt"

	"github.com/soldatov-s/accp/internal/httpsrv"
)

func main() {
	fmt.Println("Load Access Control Caching Proxy")
	httpsrv.Start()
}
