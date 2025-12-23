package main

import (
	"github.com/bios1311/tospotify/internal/apis"
	"github.com/bios1311/tospotify/internal/client"
)

func main() {
	clnt := client.CreateClient()
	err := apis.CallAPI(clnt)
	if err != nil {
		panic(err)
	}
}
