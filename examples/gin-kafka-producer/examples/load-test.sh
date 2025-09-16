#!/bin/bash

# Gin Kafka Producer - Load Testing Script
# This script performs load testing on the gin-kafka-producer service

BASE_URL="http://localhost:8080"
CONCURRENT_REQUESTS=10
TOTAL_REQUESTS=100
TEST_DURATION=60  # seconds

echo "🚀 Gin Kafka Producer - Load Testing"
echo "===================================="

# Check if service is running
echo ""
echo "1. 🔍 Service Health Check"
echo "--------------------------"
health_response=$(curl -s "$BASE_URL/health")
if [ $? -eq 0 ]; then
    echo "✅ Service is running"
    echo "$health_response" | jq .
else
    echo "❌ Service is not responding"
    exit 1
fi

# Function to send a test message
send_test_message() {
    local id=$1
    local timestamp=$(date -Iseconds)
    
    curl -s -X POST "$BASE_URL/api/v1/notifications" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"Load Test Message $id\",
            \"body\": \"This is load test message number $id sent at $timestamp\",
            \"priority\": $((1 + RANDOM % 5)),
            \"targets\": [
                {\"type\": \"email\", \"value\": \"test$id@example.com\"},
                {\"type\": \"user\", \"value\": \"user$id\", \"platform\": \"slack\"}
            ],
            \"variables\": {
                \"test_id\": \"$id\",
                \"timestamp\": \"$timestamp\",
                \"batch\": \"load_test\"
            },
            \"metadata\": {
                \"test_type\": \"load_test\",
                \"client\": \"bash_script\"
            }
        }" > /dev/null
    
    if [ $? -eq 0 ]; then
        echo "✅ Message $id sent successfully"
        return 0
    else
        echo "❌ Message $id failed"
        return 1
    fi
}

# Test 1: Sequential Load Test
echo ""
echo "2. 📊 Sequential Load Test"
echo "--------------------------"
echo "Sending $TOTAL_REQUESTS messages sequentially..."

start_time=$(date +%s)
success_count=0
error_count=0

for i in $(seq 1 $TOTAL_REQUESTS); do
    if send_test_message $i; then
        ((success_count++))
    else
        ((error_count++))
    fi
    
    # Show progress every 10 messages
    if [ $((i % 10)) -eq 0 ]; then
        echo "Progress: $i/$TOTAL_REQUESTS messages sent"
    fi
done

end_time=$(date +%s)
duration=$((end_time - start_time))
throughput=$(echo "scale=2; $TOTAL_REQUESTS / $duration" | bc)

echo ""
echo "Sequential Test Results:"
echo "  📈 Total Messages: $TOTAL_REQUESTS"
echo "  ✅ Successful: $success_count"
echo "  ❌ Failed: $error_count"
echo "  ⏱️  Duration: ${duration}s"
echo "  🚀 Throughput: ${throughput} msg/s"

# Test 2: Concurrent Load Test
echo ""
echo "3. ⚡ Concurrent Load Test"
echo "-------------------------"
echo "Sending $TOTAL_REQUESTS messages with $CONCURRENT_REQUESTS concurrent workers..."

# Create a temporary directory for worker results
temp_dir=$(mktemp -d)
trap "rm -rf $temp_dir" EXIT

# Function for concurrent worker
concurrent_worker() {
    local worker_id=$1
    local messages_per_worker=$2
    local start_id=$3
    local worker_success=0
    local worker_errors=0
    
    for i in $(seq $start_id $((start_id + messages_per_worker - 1))); do
        if send_test_message "w${worker_id}_${i}" >/dev/null 2>&1; then
            ((worker_success++))
        else
            ((worker_errors++))
        fi
    done
    
    echo "$worker_success,$worker_errors" > "$temp_dir/worker_$worker_id"
}

# Calculate messages per worker
messages_per_worker=$((TOTAL_REQUESTS / CONCURRENT_REQUESTS))
remaining_messages=$((TOTAL_REQUESTS % CONCURRENT_REQUESTS))

# Start concurrent test
start_time=$(date +%s)

# Launch workers
for worker_id in $(seq 1 $CONCURRENT_REQUESTS); do
    start_id=$(((worker_id - 1) * messages_per_worker + 1))
    worker_messages=$messages_per_worker
    
    # Give remaining messages to last worker
    if [ $worker_id -eq $CONCURRENT_REQUESTS ]; then
        worker_messages=$((messages_per_worker + remaining_messages))
    fi
    
    concurrent_worker $worker_id $worker_messages $start_id &
    echo "🏃 Worker $worker_id started (messages: $worker_messages)"
done

# Wait for all workers to complete
wait

end_time=$(date +%s)
duration=$((end_time - start_time))

# Collect results from workers
total_success=0
total_errors=0

for worker_id in $(seq 1 $CONCURRENT_REQUESTS); do
    if [ -f "$temp_dir/worker_$worker_id" ]; then
        result=$(cat "$temp_dir/worker_$worker_id")
        worker_success=$(echo $result | cut -d',' -f1)
        worker_errors=$(echo $result | cut -d',' -f2)
        total_success=$((total_success + worker_success))
        total_errors=$((total_errors + worker_errors))
        echo "Worker $worker_id: $worker_success success, $worker_errors errors"
    fi
done

throughput=$(echo "scale=2; $TOTAL_REQUESTS / $duration" | bc)

echo ""
echo "Concurrent Test Results:"
echo "  📈 Total Messages: $TOTAL_REQUESTS"
echo "  👥 Concurrent Workers: $CONCURRENT_REQUESTS"
echo "  ✅ Successful: $total_success"
echo "  ❌ Failed: $total_errors"
echo "  ⏱️  Duration: ${duration}s"
echo "  🚀 Throughput: ${throughput} msg/s"

# Test 3: Sustained Load Test
echo ""
echo "4. 🔥 Sustained Load Test"
echo "-------------------------"
echo "Sending messages continuously for $TEST_DURATION seconds..."

start_time=$(date +%s)
end_test_time=$((start_time + TEST_DURATION))
sustained_success=0
sustained_errors=0
message_id=1

while [ $(date +%s) -lt $end_test_time ]; do
    if send_test_message "sustained_$message_id" >/dev/null 2>&1; then
        ((sustained_success++))
    else
        ((sustained_errors++))
    fi
    
    ((message_id++))
    
    # Show progress every 50 messages
    if [ $((message_id % 50)) -eq 0 ]; then
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        remaining=$((TEST_DURATION - elapsed))
        echo "Sustained test: ${elapsed}s elapsed, ${remaining}s remaining (sent: $message_id messages)"
    fi
    
    # Small delay to prevent overwhelming
    sleep 0.1
done

actual_duration=$(($(date +%s) - start_time))
sustained_throughput=$(echo "scale=2; $message_id / $actual_duration" | bc)

echo ""
echo "Sustained Test Results:"
echo "  📈 Total Messages: $message_id"
echo "  ✅ Successful: $sustained_success"
echo "  ❌ Failed: $sustained_errors"
echo "  ⏱️  Duration: ${actual_duration}s"
echo "  🚀 Average Throughput: ${sustained_throughput} msg/s"

# Test 4: Service Metrics
echo ""
echo "5. 📊 Service Metrics"
echo "--------------------"
echo "Fetching final service metrics..."

metrics_response=$(curl -s "$BASE_URL/metrics")
if [ $? -eq 0 ]; then
    echo "$metrics_response" | jq .
else
    echo "❌ Failed to fetch metrics"
fi

status_response=$(curl -s "$BASE_URL/status")
if [ $? -eq 0 ]; then
    echo ""
    echo "Service Status:"
    echo "$status_response" | jq .
fi

# Summary
echo ""
echo "📋 Load Test Summary"
echo "==================="
echo "Test Scenarios Completed:"
echo "  ✅ Sequential Load Test: $throughput msg/s"
echo "  ✅ Concurrent Load Test: $throughput msg/s ($CONCURRENT_REQUESTS workers)"
echo "  ✅ Sustained Load Test: $sustained_throughput msg/s (${TEST_DURATION}s duration)"
echo ""
echo "Recommendations:"
if (( $(echo "$throughput > 50" | bc -l) )); then
    echo "  🚀 Excellent performance! Service can handle high loads."
elif (( $(echo "$throughput > 20" | bc -l) )); then
    echo "  ✅ Good performance. Consider optimizing for higher loads."
else
    echo "  ⚠️  Performance below expectations. Check system resources and configuration."
fi

echo ""
echo "Next Steps:"
echo "  1. Monitor Kafka consumer processing"
echo "  2. Check system resource usage"
echo "  3. Tune Kafka producer settings if needed"
echo "  4. Scale horizontally if required"
echo ""
echo "🎉 Load testing completed!"