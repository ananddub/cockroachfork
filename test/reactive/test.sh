#!/bin/bash

COCKROACH="/mnt/d/Devloper/cockroach/cockroach-short"

# Setup
echo "Setting up test table..."
$COCKROACH sql --insecure -e "DROP TABLE IF EXISTS test; CREATE TABLE test (id INT PRIMARY KEY, name STRING);"

# Start reactive query in background
echo "Starting reactive query..."
cd test/reactive && go run main.go &
PID=$!
cd ../..

sleep 2

# Test INSERT
echo -e "\n=== Testing INSERT ==="
$COCKROACH sql --insecure -e "INSERT INTO test VALUES (1, 'Alice');"
sleep 1

# Test UPDATE
echo -e "\n=== Testing UPDATE ==="
$COCKROACH sql --insecure -e "UPDATE test SET name = 'Bob' WHERE id = 1;"
sleep 1

# Test another INSERT
echo -e "\n=== Testing INSERT 2 ==="
$COCKROACH sql --insecure -e "INSERT INTO test VALUES (2, 'Charlie');"
sleep 1

# Test DELETE
echo -e "\n=== Testing DELETE ==="
$COCKROACH sql --insecure -e "DELETE FROM test WHERE id = 1;"
sleep 1

# Cleanup
kill $PID 2>/dev/null
echo -e "\nTest complete!"
