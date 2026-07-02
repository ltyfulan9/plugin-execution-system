package main

import (
	"context"
	"fmt"
	"log"

	"plugin-execution-system/sdk/go/pesclient"
)

func main() {
	client := pesclient.New("http://127.0.0.1:8080", "demo-token")
	plugins, err := client.ListPlugins(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range plugins {
		fmt.Printf("%s %s %s\n", p.ID, p.Name, p.Status)
	}
}
