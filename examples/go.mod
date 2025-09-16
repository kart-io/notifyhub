module github.com/kart-io/notifyhub/examples

go 1.21

require (
    github.com/gin-gonic/gin v1.9.1
    github.com/segmentio/kafka-go v0.4.47
    github.com/kart-io/notifyhub v0.0.0
    github.com/stretchr/testify v1.8.3
)

// Use local notifyhub module for development
replace github.com/kart-io/notifyhub => ../