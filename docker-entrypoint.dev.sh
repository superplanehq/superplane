#!/usr/bin/env bash

# Install workspace dependencies first
echo "Installing workspace dependencies..."
npm install

# Build UI library initially
echo "Building UI library..."
cd ui
npm install
npm run build:lib
cd ..

# Start air for hot-reloading the Golang application
echo "Starting Go application with Air..."
air &

# Start UI library in watch mode for automatic rebuilds
echo "Starting UI library watch mode..."
./dev-ui-lib.sh &

# Start vite for hot-reloading the web application
echo "Starting web application..."
cd web_src
npm run dev &
cd ..

echo "All services started. UI library changes will trigger automatic rebuilds."

# Wait for any process to exit
wait -n

# Exit with status of process that exited first
exit $?
