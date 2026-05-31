# Coding Standards

Language: Go

Version: 1.0

---

# Architecture

Use:

```text
Handler
↓
Service
↓
Repository
```

Never:

```text
Handler → Repository
```

---

# Package Rules

Each module contains:

```text
auth/
├── handler.go
├── service.go
├── repository.go
├── model.go
├── dto.go
├── errors.go
```

---

# Context

Every public method must accept:

```go
context.Context
```

Example:

```go
func (s *Service) GetUser(
    ctx context.Context,
    id uuid.UUID,
) (*User, error)
```

---

# Database

Use:

```text
sqlc
```

Never:

```text
SELECT *
```

Always select explicit columns.

---

# Error Handling

Use typed errors.

Example:

```go
var ErrUserNotFound = errors.New("user not found")
```

Avoid:

```go
errors.New("something wrong")
```

---

# Logging

Use structured logs.

Example:

```go
logger.Info().
    Str("user_id", userID.String()).
    Msg("match joined")
```

Never:

```go
fmt.Println()
```

---

# Transactions

Only Service layer can start transactions.

Repository must never create transactions.

---

# Testing

Minimum coverage:

```text
80%
```

Required:

* Service Tests
* Battle Engine Tests

---

# Naming

Interfaces:

```go
type UserRepository interface {}
```

Implementations:

```go
type PostgresUserRepository struct {}
```

---

# DTO

Request DTO

```go
LoginRequest
```

Response DTO

```go
LoginResponse
```

---

# Time

Always store:

```go
UTC
```

Never store local timezone.

---

# UUID

All primary keys:

```text
UUID v7
```

---

# Redis Keys

Pattern:

```text
module:resource:id
```

Examples:

```text
match:123:state

user:123:connection

queue:medium:100
```

---

# WebSocket Events

Format:

```json
{
  "event":"battle.started",
  "data":{}
}
```

Always include:

```text
event
data
```

---

# Pull Request Checklist

* [ ] Unit test added
* [ ] Integration test added
* [ ] No SELECT *
* [ ] No business logic in handler
* [ ] No business logic in repository
* [ ] Structured logging
* [ ] OpenTelemetry tracing
* [ ] Context propagated

```
```
