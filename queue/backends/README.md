# Queue Backends

This directory contains different queue backend implementations.

## Available Backends

### Redis Backend (`redis/`)
- Redis Streams based queue implementation
- Persistent message storage
- Consumer group support
- Suitable for production environments requiring persistence

## Adding New Backends

To add a new queue backend:

1. Create a new subdirectory (e.g., `postgres/`, `rabbitmq/`)
2. Implement the `core.Queue` interface
3. Add appropriate configuration and connection management
4. Include comprehensive tests
5. Document usage in a README.md file

## Backend Selection Guide

- **In-Memory (`core.SimpleQueue`)** - Development, testing, high-performance scenarios
- **Redis (`redis.RedisQueue`)** - Production environments requiring persistence and scalability