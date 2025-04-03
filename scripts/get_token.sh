#!/bin/bash

# Default values
EMAIL="test@example.com"
PASSWORD="password123"
AUTH_EMULATOR_HOST="localhost:9099"
API_HOST="localhost:8888"
API_KEY="fake-api-key"  # Firebase emulator accepts any value as API key

# Parse command line arguments
while [[ "$#" -gt 0 ]]; do
  case $1 in
    -e|--email) EMAIL="$2"; shift ;;
    -p|--password) PASSWORD="$2"; shift ;;
    -h|--host) AUTH_EMULATOR_HOST="$2"; shift ;;
    -a|--api) API_HOST="$2"; shift ;;
    -k|--key) API_KEY="$2"; shift ;;
    --create) CREATE_USER=1 ;;
    --test) TEST_API=1 ;;
    *) echo "Unknown parameter: $1"; exit 1 ;;
  esac
  shift
done

# Create a user if requested
if [ ! -z "$CREATE_USER" ]; then
  echo "Creating user: $EMAIL with password: $PASSWORD"
  create_response=$(curl -s -X POST "http://$AUTH_EMULATOR_HOST/identitytoolkit.googleapis.com/v1/accounts:signUp?key=$API_KEY" \
    -H 'Content-Type: application/json' \
    --data-binary "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"returnSecureToken\":true}")

  # Check if user creation was successful
  if echo "$create_response" | grep -q "idToken"; then
    echo "User created successfully"
  else
    echo "Error creating user: $create_response"
    # Continue anyway, as the user might already exist
  fi
fi

# Sign in to get the token
echo "Getting token for user: $EMAIL"
response=$(curl -s -X POST "http://$AUTH_EMULATOR_HOST/identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=$API_KEY" \
  -H 'Content-Type: application/json' \
  --data-binary "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"returnSecureToken\":true}")

# Check if we got a token
if echo "$response" | grep -q "idToken"; then
  token=$(echo "$response" | grep -o '"idToken":"[^"]*' | sed 's/"idToken":"//')
  uid=$(echo "$response" | grep -o '"localId":"[^"]*' | sed 's/"localId":"//')
  echo "Token: $token"
  echo "User UID: $uid"

  # Test the API if requested
  if [ ! -z "$TEST_API" ]; then
    echo -e "\nTesting /health endpoint:"
    curl -s "http://$API_HOST/health"

    echo -e "\n\nTesting /api/v1/users/me endpoint:"
    curl -s -H "Authorization: Bearer $token" "http://$API_HOST/api/v1/users/me" | json_pp

    echo -e "\n\nTesting /api/v1/users/$uid endpoint:"
    curl -s -H "Authorization: Bearer $token" "http://$API_HOST/api/v1/users/$uid" | json_pp
  fi
else
  echo "Error getting token: $response"
  exit 1
fi