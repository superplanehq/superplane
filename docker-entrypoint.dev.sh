#!/usr/bin/env bash

# Start air for hot-reloading the Golang application
air &

# Start vite for hot-reloading the frontend application
cd web_src
npm install
npm run dev &
cd ..

# Wait for any process to exit
wait -n

# Exit with status of process that exited first
exit $?
