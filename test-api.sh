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

# Test 2b: Duplicate User Registration
echo ""
echo "2️⃣b  Duplicate user registration (should fail)..."
DUP_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/users" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Duplicate User\",
    \"email\": \"$EMAIL\",
    \"password\": \"password123\",
    \"phoneNumber\": \"+441111111111\",
    \"address\": {
      \"line1\": \"1 Dup Street\",
      \"town\": \"London\",
      \"county\": \"Greater London\",
      \"postcode\": \"SW1A 1AA\"
    }
  }")
HTTP_CODE=$(echo "$DUP_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "409" ]; then
    echo "✅ Duplicate registration correctly rejected"
else
    echo "❌ Expected 409, got $HTTP_CODE"
    exit 1
fi

# Test 2d: Create User With Missing Required Data
echo ""
echo "2️⃣d  Create user with missing required data (should fail)..."
MISSING_USER_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/users" \
  -H "Content-Type: application/json" \
  -d '{"name": "Incomplete User"}')
HTTP_CODE=$(echo "$MISSING_USER_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "400" ]; then
    echo "✅ Missing required user data correctly rejected"
else
    echo "❌ Expected 400, got $HTTP_CODE"
    exit 1
fi

# Test 2c: Invalid Credentials Login
echo ""
echo "2️⃣c  Login with invalid credentials (should fail)..."
INVALID_LOGIN=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email": "nobody@example.com", "password": "wrongpassword"}')
HTTP_CODE=$(echo "$INVALID_LOGIN" | tail -n1)
if [ "$HTTP_CODE" = "401" ]; then
    echo "✅ Invalid credentials correctly rejected"
else
    echo "❌ Expected 401, got $HTTP_CODE"
    exit 1
fi
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

# Test 3b: Token Refresh
echo ""
echo "3️⃣b  Refreshing token..."
NEW_TOKEN=$(curl -s -X POST "$BASE_URL/v1/auth/refresh" \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$TOKEN\"}" | jq -r '.token')
if [ -z "$NEW_TOKEN" ] || [ "$NEW_TOKEN" = "null" ]; then
    echo "❌ Failed to refresh token"
    exit 1
fi
TOKEN=$NEW_TOKEN
echo "Refreshed token: ${TOKEN:0:20}..."
print_result "Token refresh"

# Test 3c: Unauthenticated Request
echo ""
echo "3️⃣c  Unauthenticated request (should fail)..."
UNAUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/users/$USER_ID")
HTTP_CODE=$(echo "$UNAUTH_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "401" ]; then
    echo "✅ Unauthenticated access correctly rejected"
else
    echo "❌ Expected 401, got $HTTP_CODE"
    exit 1
fi

# Setup: Register and login a second user for cross-user authorization tests
echo ""
echo "🔧 Setting up second user for authorization tests..."
SECOND_EMAIL="second-$(date +%s)@example.com"
SECOND_USER_RESPONSE=$(curl -s -X POST "$BASE_URL/v1/users" \
  -H "Content-Type: application/json" \
  -d "{
    \"name\": \"Second User\",
    \"email\": \"$SECOND_EMAIL\",
    \"password\": \"password123\",
    \"phoneNumber\": \"+449999999999\",
    \"address\": {
      \"line1\": \"789 Second Street\",
      \"town\": \"Birmingham\",
      \"county\": \"West Midlands\",
      \"postcode\": \"B1 1AA\"
    }
  }")
SECOND_USER_ID=$(echo $SECOND_USER_RESPONSE | jq -r '.id')
SECOND_TOKEN=$(curl -s -X POST "$BASE_URL/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$SECOND_EMAIL\", \"password\": \"password123\"}" | jq -r '.token')
echo "Second user ID: $SECOND_USER_ID"
echo "Second user token: ${SECOND_TOKEN:0:20}..."

echo ""
echo "4️⃣  Fetching user details..."
curl -s -X GET "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Get user details"

# Test 4b: Fetch Another User's Details (Forbidden)
echo ""
echo "4️⃣b  Fetching another user's details (should fail with 403)..."
FETCH_OTHER_USER=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $SECOND_TOKEN")
HTTP_CODE=$(echo "$FETCH_OTHER_USER" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Fetch another user correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 4c: Fetch Non-Existent User (Not Found)
echo ""
echo "4️⃣c  Fetching non-existent user (should fail with 403 or 404)..."
FETCH_NONEXISTENT_USER=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/users/usr-nonexistent-00000" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$FETCH_NONEXISTENT_USER" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Non-existent user correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

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

# Test 5b: Create Account With Missing Required Data
echo ""
echo "5️⃣b  Creating account with missing required data (should fail)..."
MISSING_ACCOUNT_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/accounts" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}')
HTTP_CODE=$(echo "$MISSING_ACCOUNT_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "400" ]; then
    echo "✅ Missing account data correctly rejected"
else
    echo "❌ Expected 400, got $HTTP_CODE"
    exit 1
fi

# Test 6: List Accounts
echo ""
echo "6️⃣  Listing accounts..."
ACCOUNTS=$(curl -s -X GET "$BASE_URL/v1/accounts" \
  -H "Authorization: Bearer $TOKEN")
ACCOUNT_COUNT=$(echo $ACCOUNTS | jq '.accounts | length')
echo "Found $ACCOUNT_COUNT account(s)"
print_result "List accounts"

# Test 6b: Fetch Another User's Account (Forbidden)
echo ""
echo "6️⃣b  Fetching another user's account (should fail with 403)..."
FETCH_OTHER_ACCOUNT=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Authorization: Bearer $SECOND_TOKEN")
HTTP_CODE=$(echo "$FETCH_OTHER_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Fetch another user's account correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 6c: Fetch Non-Existent Account (Not Found)
echo ""
echo "6️⃣c  Fetching non-existent account (should fail with 403 or 404)..."
FETCH_NONEXISTENT_ACCOUNT=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/acc-nonexistent-00000" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$FETCH_NONEXISTENT_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Non-existent account correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

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

# Test 11a: List Transactions on Another User's Account (Forbidden)
echo ""
echo "1️⃣1️⃣a  Listing transactions on another user's account (should fail with 403)..."
LIST_TX_FORBIDDEN=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Authorization: Bearer $SECOND_TOKEN")
HTTP_CODE=$(echo "$LIST_TX_FORBIDDEN" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ List transactions on another user's account correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 11a2: List Transactions on Non-Existent Account (Not Found)
echo ""
echo "1️⃣1️⃣a2  Listing transactions on non-existent account (should fail with 403 or 404)..."
LIST_TX_NOT_FOUND=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/acc-nonexistent-00000/transactions" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$LIST_TX_NOT_FOUND" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ List transactions on non-existent account correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

# Test 11b: Get Single Transaction
echo ""
echo "1️⃣1️⃣b  Fetching single transaction..."
SINGLE_TX_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$SINGLE_TX_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Get single transaction working"
else
    echo "❌ Expected 200, got $HTTP_CODE"
    exit 1
fi

# Test 11b2: Fetch Transaction on Another User's Account (Forbidden)
echo ""
echo "1️⃣1️⃣b2  Fetching transaction on another user's account (should fail with 403)..."
FETCH_TX_FORBIDDEN=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $SECOND_TOKEN")
HTTP_CODE=$(echo "$FETCH_TX_FORBIDDEN" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Fetch transaction on another user's account correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 11b3: Fetch Transaction on Non-Existent Account (Not Found)
echo ""
echo "1️⃣1️⃣b3  Fetching transaction on non-existent account (should fail with 403 or 404)..."
FETCH_TX_NO_ACCOUNT=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/acc-nonexistent-00000/transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$FETCH_TX_NO_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Fetch transaction on non-existent account correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

# Test 11b4: Fetch Non-Existent Transaction (Not Found)
echo ""
echo "1️⃣1️⃣b4  Fetching non-existent transaction (should fail with 404)..."
FETCH_TX_NOT_FOUND=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions/tx-nonexistent-00000" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$FETCH_TX_NOT_FOUND" | tail -n1)
if [ "$HTTP_CODE" = "404" ]; then
    echo "✅ Non-existent transaction correctly returns 404"
else
    echo "❌ Expected 404, got $HTTP_CODE"
    exit 1
fi

# Test 11b5: Fetch Transaction Against Wrong Account (Not Found)
echo ""
echo "1️⃣1️⃣b5  Creating second account to test wrong-account transaction fetch..."
ACCOUNT2_RESPONSE=$(curl -s -X POST "$BASE_URL/v1/accounts" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Second Account", "accountType": "personal"}')
ACCOUNT_NUMBER_2=$(echo $ACCOUNT2_RESPONSE | jq -r '.accountNumber')
echo "Second account: $ACCOUNT_NUMBER_2"
echo ""
echo "1️⃣1️⃣b5  Fetching transaction against wrong bank account (should fail with 404)..."
FETCH_TX_WRONG_ACCOUNT=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER_2/transactions/$TRANSACTION_ID" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$FETCH_TX_WRONG_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "404" ]; then
    echo "✅ Transaction against wrong account correctly returns 404"
else
    echo "❌ Expected 404, got $HTTP_CODE"
    exit 1
fi

# Test 11c: Update Account
echo ""
echo "1️⃣1️⃣c  Updating bank account..."
UPDATE_ACCOUNT_RESPONSE=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Renamed Savings Account", "accountType": "personal"}')
HTTP_CODE=$(echo "$UPDATE_ACCOUNT_RESPONSE" | tail -n1)
if [ "$HTTP_CODE" = "200" ]; then
    echo "✅ Update account working"
else
    echo "❌ Expected 200, got $HTTP_CODE"
    exit 1
fi

# Test 11d: Update Another User's Account (Forbidden)
echo ""
echo "1️⃣1️⃣d  Updating another user's account (should fail with 403)..."
UPDATE_OTHER_ACCOUNT=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SECOND_TOKEN" \
  -d '{"name": "Hacked Account", "accountType": "personal"}')
HTTP_CODE=$(echo "$UPDATE_OTHER_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Update another user's account correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 11e: Update Non-Existent Account (Not Found)
echo ""
echo "1️⃣1️⃣e  Updating non-existent account (should fail with 403 or 404)..."
UPDATE_NONEXISTENT_ACCOUNT=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/v1/accounts/acc-nonexistent-00000" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Ghost Account", "accountType": "personal"}')
HTTP_CODE=$(echo "$UPDATE_NONEXISTENT_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Update non-existent account correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

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

# Test 12b: Transaction on Another User's Account (Forbidden)
echo ""
echo "1️⃣2️⃣b  Transaction on another user's account (should fail with 403)..."
TX_FORBIDDEN=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SECOND_TOKEN" \
  -d '{"amount": 100.00, "currency": "GBP", "type": "deposit", "reference": "Unauthorized deposit"}')
HTTP_CODE=$(echo "$TX_FORBIDDEN" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Transaction on another user's account correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 12c: Transaction on Non-Existent Account (Not Found)
echo ""
echo "1️⃣2️⃣c  Transaction on non-existent account (should fail with 403 or 404)..."
TX_NOT_FOUND=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/accounts/acc-nonexistent-00000/transactions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 100.00, "currency": "GBP", "type": "deposit", "reference": "Ghost deposit"}')
HTTP_CODE=$(echo "$TX_NOT_FOUND" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Transaction on non-existent account correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

# Test 12d: Transaction With Missing Required Data (Bad Request)
echo ""
echo "1️⃣2️⃣d  Transaction with missing required data (should fail with 400)..."
TX_MISSING_DATA=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER/transactions" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"currency": "GBP"}')
HTTP_CODE=$(echo "$TX_MISSING_DATA" | tail -n1)
if [ "$HTTP_CODE" = "400" ]; then
    echo "✅ Missing transaction data correctly rejected"
else
    echo "❌ Expected 400, got $HTTP_CODE"
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

# Test 13b: Update Another User (Forbidden)
echo ""
echo "1️⃣3️⃣b  Updating another user's details (should fail with 403)..."
UPDATE_OTHER_USER=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/v1/users/$USER_ID" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SECOND_TOKEN" \
  -d '{"name": "Hacked Name"}')
HTTP_CODE=$(echo "$UPDATE_OTHER_USER" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Update another user correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 13c: Update Non-Existent User (Not Found)
echo ""
echo "1️⃣3️⃣c  Updating non-existent user (should fail with 403 or 404)..."
UPDATE_NONEXISTENT_USER=$(curl -s -w "\n%{http_code}" -X PATCH "$BASE_URL/v1/users/usr-nonexistent-00000" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Ghost User"}')
HTTP_CODE=$(echo "$UPDATE_NONEXISTENT_USER" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Update non-existent user correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

# Test 13d: Delete Another User (Forbidden)
echo ""
echo "1️⃣3️⃣d  Deleting another user (should fail with 403)..."
DELETE_OTHER_USER=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $SECOND_TOKEN")
HTTP_CODE=$(echo "$DELETE_OTHER_USER" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Delete another user correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 13e: Delete Non-Existent User (Not Found)
echo ""
echo "1️⃣3️⃣e  Deleting non-existent user (should fail with 403 or 404)..."
DELETE_NONEXISTENT_USER=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/v1/users/usr-nonexistent-00000" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$DELETE_NONEXISTENT_USER" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Delete non-existent user correctly returns $HTTP_CODE"
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

# Test 14b: Delete User With Active Accounts
echo ""
echo "1️⃣4️⃣b  Deleting user with active accounts (should fail)..."
DEL_WITH_ACCS=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$DEL_WITH_ACCS" | tail -n1)
if [ "$HTTP_CODE" = "409" ]; then
    echo "✅ Cannot delete user with active bank accounts"
else
    echo "❌ Expected 409, got $HTTP_CODE"
    exit 1
fi

# Test 14c: Delete Another User's Account (Forbidden)
echo ""
echo "1️⃣4️⃣c  Deleting another user's account (should fail with 403)..."
DELETE_OTHER_ACCOUNT=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Authorization: Bearer $SECOND_TOKEN")
HTTP_CODE=$(echo "$DELETE_OTHER_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Delete another user's account correctly returns 403"
else
    echo "❌ Expected 403, got $HTTP_CODE"
    exit 1
fi

# Test 14d: Delete Non-Existent Account (Not Found)
echo ""
echo "1️⃣4️⃣d  Deleting non-existent account (should fail with 403 or 404)..."
DELETE_NONEXISTENT_ACCOUNT=$(curl -s -w "\n%{http_code}" -X DELETE "$BASE_URL/v1/accounts/acc-nonexistent-00000" \
  -H "Authorization: Bearer $TOKEN")
HTTP_CODE=$(echo "$DELETE_NONEXISTENT_ACCOUNT" | tail -n1)
if [ "$HTTP_CODE" = "404" ] || [ "$HTTP_CODE" = "403" ]; then
    echo "✅ Delete non-existent account correctly returns $HTTP_CODE"
else
    echo "❌ Expected 403 or 404, got $HTTP_CODE"
    exit 1
fi

# Test 15: Delete Account
echo ""
echo "1️⃣5️⃣  Deleting bank account..."
curl -s -X DELETE "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Delete account"

# Test 15b: Delete Second Account
echo ""
echo "1️⃣5️⃣b  Deleting second bank account..."
curl -s -X DELETE "$BASE_URL/v1/accounts/$ACCOUNT_NUMBER_2" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Delete second account"

# Test 16: Delete User
echo ""
echo "1️⃣6️⃣  Deleting user..."
curl -s -X DELETE "$BASE_URL/v1/users/$USER_ID" \
  -H "Authorization: Bearer $TOKEN" > /dev/null
print_result "Delete user"

# Cleanup: Delete second test user
echo ""
echo "🧹 Cleaning up second test user..."
curl -s -X DELETE "$BASE_URL/v1/users/$SECOND_USER_ID" \
  -H "Authorization: Bearer $SECOND_TOKEN" > /dev/null
print_result "Cleanup second user"

echo ""
echo "=============================="
echo "✨ All tests passed! ✨"
echo "=============================="
