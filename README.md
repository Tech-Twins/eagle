# Quick Start

Assumes Docker Desktop is running.

## 1. Configure environment

```bash
cp .env.example .env
```

Open `.env` and set `JWT_SECRET` to something non-trivial before proceeding.

Alternatively, export the variable directly in your shell (no `.env` file required):

```bash
export JWT_SECRET=your-secret-key-here
```

## 2. Start everything

```bash
bash setup.sh
```

Or manually:
```bash
docker-compose up --build
```

Wait for all containers to report healthy (~30s). You can check with:
```bash
curl http://localhost:8080/health
```

## 3. Create a user

```bash
curl -s -X POST http://localhost:8080/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Alice Smith",
    "email": "alice@example.com",
    "password": "letmein123",
    "phoneNumber": "+441234567890",
    "address": {
      "line1": "10 Downing Street",
      "town": "London",
      "county": "Greater London",
      "postcode": "SW1A 2AA"
    }
  }' | jq .
```

Note the `id` field in the response — you'll need it later.

## 4. Log in

```bash
export TOKEN=$(curl -s -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"letmein123"}' | jq -r '.token')
```

## 5. Open a bank account

```bash
export ACCOUNT=$(curl -s -X POST http://localhost:8080/v1/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Current","accountType":"personal"}' | jq -r '.accountNumber')

echo "Account number: $ACCOUNT"
```

## 6. Deposit

```bash
curl -s -X POST http://localhost:8080/v1/accounts/$ACCOUNT/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "amount": 1000.00,
    "currency": "GBP",
    "type": "deposit",
    "reference": "Opening deposit"
  }' | jq .
```

Balance updates are event-driven — give it a second or two before checking.

## 7. Check balance

```bash
curl -s http://localhost:8080/v1/accounts/$ACCOUNT \
  -H "Authorization: Bearer $TOKEN" | jq .balance
```

## Try a withdrawal

```bash
curl -s -X POST http://localhost:8080/v1/accounts/$ACCOUNT/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount":100.00,"currency":"GBP","type":"withdrawal","reference":"cash"}' | jq .
```

Attempting to withdraw more than the balance returns a `422`.

## Run the automated test suite

```bash
bash test-api.sh
```

## Teardown

```bash
docker-compose down        # stop, keep volumes
docker-compose down -v     # stop and wipe all data
```

## Troubleshooting

**Services not coming up:** `docker-compose logs` is your friend. Postgres health checks can occasionally time out on slower machines — just re-run `docker-compose up`.

**Reset state mid-test:** `docker-compose down -v && bash setup.sh`

**Connect to a DB directly:**
```bash
docker exec -it eagle-postgres-users psql -U postgres -d eagle_users
```
