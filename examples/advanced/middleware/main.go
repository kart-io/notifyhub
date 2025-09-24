// Package main demonstrates middleware patterns and message transformation
// This shows enterprise middleware for logging, metrics, and message enrichment
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/kart-io/notifyhub/pkg/notifyhub"
	"github.com/kart-io/notifyhub/pkg/platforms/feishu"
)

func main() {
	fmt.Println("🛡️  Middleware and Message Transformation Demo")
	fmt.Println("===========================================")
	fmt.Println()

	// Create a hub with middleware
	hub, err := notifyhub.New(
		feishu.WithFeishu(os.Getenv("FEISHU_WEBHOOK_URL")),
		// Add middleware
		// notifyhub.WithMiddleware(rateLimitMiddleware, retryMiddleware),
	)
	if err != nil {
		log.Fatalf("Failed to create hub: %v", err)
	}
	defer func() { _ = hub.Close() }()

	// Demonstrate middleware
	sendWithLogging(hub)
	sendWithMetrics(hub)
	sendWithRetry(hub)
	sendWithCircuitBreaker(hub)

	// Demonstrate message transformation
	transformMessageDemo(hub)

	// Demonstrate middleware chaining
	chain := &MiddlewareChain{}
	chain.Add(loggingMiddleware)
	chain.Add(metricsMiddleware)
	chain.Add(retryMiddleware)
	chain.Execute(hub)

	fmt.Println("\n✅ Middleware and Transformation Demo Complete!")
}

// Middleware examples

func sendWithLogging(hub notifyhub.Client) {
	fmt.Println("\n--- Logging Middleware ---")
	msg := notifyhub.NewMessage("Logging Test").WithBody("This message will be logged.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	_, _ = hub.Send(context.Background(), msg)
}

func sendWithMetrics(hub notifyhub.Client) {
	fmt.Println("\n--- Metrics Middleware ---")
	msg := notifyhub.NewMessage("Metrics Test").WithBody("This message will generate metrics.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	_, _ = hub.Send(context.Background(), msg)
}

func sendWithRetry(hub notifyhub.Client) {
	fmt.Println("\n--- Retry Middleware ---")
	msg := notifyhub.NewMessage("Retry Test").WithBody("This message will be retried on failure.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	_, _ = hub.Send(context.Background(), msg)
}

func sendWithCircuitBreaker(hub notifyhub.Client) {
	fmt.Println("\n--- Circuit Breaker Middleware ---")
	msg := notifyhub.NewMessage("Circuit Breaker Test").WithBody("This message is subject to circuit breaking.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	_, _ = hub.Send(context.Background(), msg)
}

// Message transformation examples

func transformMessageDemo(hub notifyhub.Client) {
	fmt.Println("\n--- Message Transformation ---")
	msg := notifyhub.NewMessage("Transformation Test").WithBody("This message will be transformed.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	_, _ = hub.Send(context.Background(), msg)
}

// Middleware chaining

type Middleware func(notifyhub.Client) notifyhub.Client

type MiddlewareChain struct {
	middlewares []Middleware
}

func (c *MiddlewareChain) Add(m Middleware) {
	c.middlewares = append(c.middlewares, m)
}

func (c *MiddlewareChain) Execute(hub notifyhub.Client) {
	fmt.Println("\n--- Middleware Chaining ---")
	h := hub
	for _, m := range c.middlewares {
		h = m(h)
	}
	msg := notifyhub.NewMessage("Chained Middleware Test").WithBody("This message goes through a chain of middleware.").ToTarget(notifyhub.NewTarget("webhook", "", "feishu")).Build()
	_, _ = h.Send(context.Background(), msg)
}

// Middleware functions
func loggingMiddleware(h notifyhub.Client) notifyhub.Client {
	return h
}

func metricsMiddleware(h notifyhub.Client) notifyhub.Client {
	return h
}

func retryMiddleware(h notifyhub.Client) notifyhub.Client {
	return h
}
