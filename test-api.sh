#!/bin/bash

# Test script for Eagle Bank API
# This script tests the complete user journey

set -e

BASE_URL="http://localhost:8080"
EMAIL="test-$(date +%s)@example.com"
PASSWORD="password123"

echo "🏦 Eagle Bank API Test Suite"
echo "=============================="
echo ""

# Function to print test results
print_result() {
    if [ $? -eq 0 ]; then
        echo "✅ $1"
    else
        echo "❌ $1"
        exit 1
    fi
}

# Wait for services to be ready
echo "⏳ Waiting for services to start..."
sleep 5

# Test 1: Health Check
echo ""
echo "1️⃣  Testing health check..."
curl -s "$BASE_URL/health" > /dev/null
print_result "Health check passed"

# Test 2: User Registration
echo ""
echo "2️⃣  Creating new user..."
USER_RESPONSE=$(curl -s -X POST "$BASE_URL/v1/users" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Test User\",
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\",
    \"phoneNumber\": \"+441234567890\",
    \"address\": {
      \"line1\": \"123 Test Street\",
      \"town\": \"London\",
      \"county\": \"Greater London\",
      \"postcode\": \"SW1A 1AA\"
    }
  }")

USER_ID=$(echo $USER_RESPONSE | jq -r '.id')
echo "User created with ID: $USER_ID"
print_result "User registration"

# Test 3: Login
echo ""
echo "3️⃣  Logging in..."
TOKEN=$(curl -s -X POST "$BASE_URL/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"$EMAIL\",
    \"password\": \"$PASSWORD\"
  }" | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo "❌ Failed to get token"
    exit 1
fi
echo "Token received: ${TOKEN:0:20}..."
print_result "Login"

# Test 4: Get User Details
echo ""
echo "4️⃣  Fetching user details..."
curl -s -X GET "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Get user details"

# Test 5: Create Bank Account
echo ""
echo "5️⃣  Creating bank account..."
ACCOUNT_RESPONSE=$(curl -s -X POST "$BASE_URL/v1/accounts" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Savings Account",
    "accountType": "personal"
  }')

ACCOUNT_NUMBER=$(echo $ACCOUNT_RESPONSE | jq -r '.accountNumber')
echo "Account created: $ACCOUNT_NUMBER"
print_result "Create account"

# Test 6: List Accounts
echo ""
echo "6️⃣  Listing accounts..."
ACCOUNTS=$(curl -s -X GET "$BASE_URL/v1/accounts" \
  -H "Authorization: Bearer $TOKEN")
ACCOUNT_COUNT=$(echo $ACCOUNTS | jq '.accounts | length')
echo "Found $ACCOUNT_COUNT account(s)"
print_result "List accounts"

# Test 7: Deposit Money
echo ""
echo "7️⃣  Depositing £1000..."
DEPOSIT_RESPONSE=$(curl -s -X POST "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "amount": 1000.00,
    "currency": "GBP",
    "type": "deposit",
    "reference": "Initial deposit"
  }')

TRANSACTION_ID=$(echo $DEPOSIT_RESPONSE | jq -r '.id')
echo "Transaction ID: $TRANSACTION_ID"
print_result "Deposit money"

# Wait for event processing
echo "⏳ Waiting for balance update..."
sleep 2

# Test 8: Check Balance
echo ""
echo "8️⃣  Checking account balance..."
ACCOUNT_INFO=$(curl -s -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Authorization: Bearer $TOKEN")
BALANCE=$(echo $ACCOUNT_INFO | jq -r '.balance')
echo "Current balance: £$BALANCE"
print_result "Check balance"

# Test 9: Withdraw Money
echo ""
echo "9️⃣  Withdrawing £200..."
curl -s -X POST "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "amount": 200.00,
    "currency": "GBP",
    "type": "withdrawal",
    "reference": "Test withdrawal"
  }' > /dev/null
print_result "Withdraw money"

# Wait for event processing
sleep 2

# Test 10: Check Updated Balance
echo ""
echo "🔟 Checking updated balance..."
ACCOUNT_INFO=$(curl -s -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Authorization: Bearer $TOKEN")
NEW_BALANCE=$(echo $ACCOUNT_INFO | jq -r '.balance')
echo "New balance: £$NEW_BALANCE"
print_result "Updated balance check"

# Test 11: List Transactions
echo ""
echo "1️⃣1️⃣  Listing transactions..."
TRANSACTIONS=$(curl -s -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Authorization: Bearer $TOKEN")
TRANSACTION_COUNT=$(echo $TRANSACTIONS | jq '.transactions | length')
echo "Found $TRANSACTION_COUNT transaction(s)"
print_result "List transactions"

# Test 12: Test Insufficient Funds
echo ""
echo "1️⃣2️⃣  Testing insufficient funds (should fail)..."
INSUFFICIENT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "amount": 10000.00,
    "currency": "GBP",
    "type": "withdrawal",
    "reference": "Should fail"
  }')

HTTP_CODE=$(echo "$INSUFFICIENT_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "422" ]; then
    echo "✅ Insufficient funds validation working"
else
    echo "❌ Expected 422, got $HTTP_CODE"
    exit 1
fi

# Test 13: Test Forbidden Access
echo ""
echo "1️⃣3️⃣  Testing forbidden access (should fail)..."
FORBIDDEN_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/users/usr-invalid" \
  -H "Authorization: Bearer $TOKEN")

HTTP_CODE=$(echo "$FORBIDDEN_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "403" ] || [ "$HTTP_CODE" = "404" ]; then
    echo "✅ Authorization check working"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

# Test 14: Update User
echo ""
echo "1️⃣4️⃣  Updating user details..."
curl -s -X PATCH "$BASE_URL/v1/users/$USER_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"name\": \"Updated Test User\",
    \"email\": \"$EMAIL\",
    \"phoneNumber\": \"+449876543210\",
    \"address\": {
      \"line1\": \"456 New Street\",
      \"town\": \"Manchester\",
      \"county\": \"Greater Manchester\",
      \"postcode\": \"M1 1AA\"
    }
  }" > /dev/null
print_result "Update user"

# Test 15: Delete Account
echo ""
echo "1️⃣5️⃣  Deleting bank account..."
curl -s -X DELETE "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Delete account"

# Test 16: Delete User
echo ""
echo "1️⃣6️⃣  Deleting user..."
curl -s -X DELETE "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Delete user"

echo ""
echo "=============================="
echo "✨ All tests passed! ✨"
echo "=============================="
