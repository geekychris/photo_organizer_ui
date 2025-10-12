#!/bin/bash

echo "Testing thumbnail generation..."

# Start the server in background
./photodatabase -serve &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Test thumbnail generation for first 3 files
echo "Testing thumbnails for first 3 files:"
for i in {1..3}; do
    echo "Testing file ID $i..."
    response=$(curl -s -w "%{http_code}" -o /dev/null http://localhost:8080/thumbnail/$i)
    if [ "$response" = "200" ]; then
        echo "✓ Thumbnail $i: SUCCESS"
    else
        echo "✗ Thumbnail $i: FAILED (HTTP $response)"
    fi
done

# Check how many thumbnails were generated
thumbnail_count=$(ls -1 thumbnails/ | wc -l)
echo ""
echo "Total thumbnails generated: $thumbnail_count"
echo "Thumbnail files:"
ls -la thumbnails/

# Stop the server
kill $SERVER_PID

echo ""
echo "Test completed. You can now start the server with: ./photodatabase -serve"
echo "Then open http://localhost:8080 in your browser"