#!/bin/bash

#
# Development script to watch and rebuild UI library automatically
#

echo "Starting UI library watch mode..."

cd ui

echo "Cleaning up any existing processes..."
pkill -f "BUILD_MODE=lib.*vite.*build.*watch" || true
pkill -f "tsc.*tsconfig.lib.json.*-w" || true

echo "Performing initial build..."
npm run build:lib

echo "Starting UI library build in watch mode..."
BUILD_MODE=lib VITE_FORCE_POLLING=true ./node_modules/.bin/vite build --watch &
VITE_PID=$!

BUILD_MODE=lib ./node_modules/.bin/tsc -p tsconfig.lib.json -w --preserveWatchOutput &
TSC_PID=$!

echo "UI library watch mode started (Vite PID: $VITE_PID, TypeScript PID: $TSC_PID)"
echo "The UI library will rebuild automatically when you save files."
echo "You can now run 'npm run dev' in the web_src directory to start development."
echo ""
echo "To stop the UI library watch mode, press Ctrl+C"

cleanup() {
    echo ""
    echo "Shutting down UI library watch mode..."
    kill $VITE_PID 2>/dev/null || true
    kill $TSC_PID 2>/dev/null || true
    exit 0
}

trap cleanup EXIT INT TERM

wait