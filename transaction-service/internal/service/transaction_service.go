// Package service is retained for Go module compatibility.
// All transaction business logic has been decomposed into the CQRS packages:
//   - internal/command  — TransactionCommandService (writes)
//   - internal/query    — TransactionQueryService   (reads from Redis read model)
package service
