#!/usr/bin/env bash

spin_chars=(
  "⠋"
  "⠙"
  "⠹"
  "⠸"
  "⠼"
  "⠴"
  "⠦"
  "⠧"
  "⠇"
  "⠏"
)
spin_index=0
SPINNER_PID=""
SPINNER_MESSAGE=""

start_spinner() {
  local message="$1"
  local spin_index=0
  SPINNER_MESSAGE="$message"

  (
      while true; do
          spin_index=$(( (spin_index + 1) % ${#spin_chars[@]} ))
          # Use printf for reliable carriage return handling
          printf "\r%s %s" "${spin_chars[$spin_index]}" "${message}"
          sleep 0.1
      done
  ) &
  
  SPINNER_PID=$!
  trap 'kill $SPINNER_PID >/dev/null 2>&1' EXIT
}

stop_spinner() {
  kill $SPINNER_PID >/dev/null 2>&1
  # Use printf to clear the line completely (ANSI escape code \033[K)
  printf "\r\033[K"
  echo "⠸ $SPINNER_MESSAGE"
  trap - EXIT
}

