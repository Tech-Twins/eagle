// Package service is retained for Go module compatibility.
// All account business logic has been decomposed into the CQRS packages:
//   - internal/command  — AccountCommandService (writes + event handling)
//   - internal/query    — AccountQueryService   (reads from Redis read model)
package service
