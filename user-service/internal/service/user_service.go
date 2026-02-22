// Package service is retained for Go module compatibility.
// All user business logic has been decomposed into the CQRS packages:
//   - internal/command  — UserCommandService (writes + event handling)
//   - internal/query    — UserQueryService   (reads from Redis read model)
package service
