# Callback System

This package provides callback functionality for queue message processing events.

## Features

- Function callbacks for all message processing events
- HTTP webhook callbacks with HMAC signature support
- Built-in logging callback implementation
- Asynchronous callback execution with timeout control

## Callback Events

- `CallbackEventSent` - Message sent successfully
- `CallbackEventFailed` - Message sending failed
- `CallbackEventRetry` - Message is being retried
- `CallbackEventMaxRetries` - Maximum retries exceeded

## Usage

```go
import "github.com/kart-io/notifyhub/queue/callbacks"

// Create function callback
successCallback := callbacks.NewCallbackFunc("success", func(ctx context.Context, callbackCtx *callbacks.CallbackContext) error {
    log.Printf("Message %s sent successfully", callbackCtx.MessageID)
    return nil
})

// Configure message callbacks
callbackOptions := &callbacks.CallbackOptions{}
callbackOptions.AddCallback(callbacks.CallbackEventSent, successCallback)

// Or use webhook callbacks
callbackOptions := &callbacks.CallbackOptions{
    WebhookURL: "https://example.com/webhook",
    WebhookSecret: "secret-key",
}
```