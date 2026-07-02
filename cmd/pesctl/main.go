package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"plugin-execution-system/sdk/go/pesclient"
)

func main() {
	base := flag.String("addr", env("PES_ADDR", "http://127.0.0.1:8080"), "PES server base URL")
	token := flag.String("token", env("PES_TOKEN", "demo-token"), "API token")
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
		os.Exit(2)
	}
	client := pesclient.New(*base, *token)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var out any
	var err error
	switch flag.Arg(0) {
	case "health":
		out, err = client.Health(ctx)
	case "plugins":
		out, err = client.ListPlugins(ctx)
	case "execution":
		if flag.NArg() < 2 {
			usage()
			os.Exit(2)
		}
		out, err = client.GetExecution(ctx, flag.Arg(1))
	case "webhooks":
		out, err = client.ListWebhooks(ctx)
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: pesctl [-addr http://127.0.0.1:8080] [-token demo-token] <health|plugins|execution ID|webhooks>")
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
