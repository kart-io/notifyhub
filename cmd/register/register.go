package main

import (
	"log"

	"github.com/kart-io/notifyhub/platforms/email"
	"github.com/kart-io/notifyhub/platforms/feishu"
	"github.com/kart-io/notifyhub/platforms/registry"
)

func init() {
	// Register built-in platforms
	if err := registry.Register(feishu.NewFeishuPlatform()); err != nil {
		log.Fatalf("failed to register feishu platform: %v", err)
	}
	if err := registry.Register(email.NewEmailPlatform()); err != nil {
		log.Fatalf("failed to register email platform: %v", err)
	}
}

func main() {
	// This is a registration utility
	// Platform registration happens in init()
}
