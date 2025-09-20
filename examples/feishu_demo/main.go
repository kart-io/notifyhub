package main

import (
	"context"
	"log"

	"github.com/kart-io/notifyhub"

	"github.com/kart-io/notifyhub/platforms/email"
	"github.com/kart-io/notifyhub/platforms/feishu"
	"github.com/kart-io/notifyhub/platforms/registry"
)

func main() {
	feishuDemo()
}

func init() {
	// Register built-in platforms
	if err := registry.Register(feishu.NewFeishuPlatform()); err != nil {
		log.Fatalf("failed to register feishu platform: %v", err)
	}
	if err := registry.Register(email.NewEmailPlatform()); err != nil {
		log.Fatalf("failed to register email platform: %v", err)
	}
}

func feishuDemo() {

	client, err := notifyhub.New(
		notifyhub.WithFeishu("https://open.feishu.cn/open-apis/bot/v2/hook/688dc0bf-c74b-41d1-a6b9-8cb660477488", "gQURr67BPOsTZlI7jBn0Jh"),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	client.Send(ctx).
		Title("Hello World").
		Body("This is a test notification").
		ToFeishu("webhook-id").
		Execute()
}
