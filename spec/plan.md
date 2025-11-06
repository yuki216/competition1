# /speckit.plan — Fixora (AI-assisted IT Ticketing)

> Versi: 1.0
> Basis: speckit constitution (DDD, Clean Architecture, TDD)

## Ringkasan

Dokumen ini adalah **plan** teknis untuk implementasi Fixora — platform IT Ticketing yang dibantu AI. Dokumen mengikat domain model, ports, use cases, delivery adapters, dan cadangan infrastruktur. Ini ditulis agar langsung jadi blueprint pengembangan (kode + testable contracts).

---

## 1. Scope & Goals

* Implementasi minimum viable product (MVP) meliputi: ticket submission (manual + AI intake), comment flow, assignment, resolve flow, simple knowledge base CRUD, AI suggestion port (mockable), dan reporting dasar.
* Kriteria keberhasilan MVP: create → assign → resolve loop lengkap dengan audit + AI training hook.

---

## 2. Bounded Contexts & Responsibilities

* **Ticket Management**: lifecycle tiket, assignment, SLA status.
* **Commenting & Collaboration**: penyimpanan komentar, real-time notifications.
* **Knowledge Base**: CRUD entri solusi & tags, versioning ringan.
* **AI Assistant**: AISuggestionService (ingest description → suggestion + confidence), AITrainingService (learn from resolved tiket).
* **Reporting**: aggregasi metric (resolution time, first response, AI accuracy).

---

## 3. Domain Model (Ringkas)

### Ticket (Aggregate Root)

* id: UUID
* title: string
* description: string
* status: OPEN | IN_PROGRESS | RESOLVED | CLOSED
* category: NETWORK | SOFTWARE | HARDWARE | ACCOUNT | OTHER
* priority: LOW | MEDIUM | HIGH | CRITICAL
* createdBy: EmployeeID
* assignedTo: AdminID | null
* aiInsight: { text: string, confidence: float } | null
* createdAt, updatedAt

### Comment

* id, ticketId, authorId, role: EMPLOYEE|ADMIN|AI, body, createdAt

### KnowledgeBaseEntry

* id, contextLabel, solutionSteps, tags[], source: MANUAL|LEARNED, active:boolean, updatedAt

### Value Objects

* Metric (resolutionTime, firstResponseTime), AuditEntry

---

## 4. Ports (Interfaces) — contoh Go signatures

```go
// repository ports
type TicketRepository interface {
  Create(ctx context.Context, t *Ticket) error
  FindByID(ctx context.Context, id string) (*Ticket, error)
  Update(ctx context.Context, t *Ticket) error
  List(ctx context.Context, filter TicketFilter) ([]*Ticket, error)
}

type CommentRepository interface {
  Create(ctx context.Context, c *Comment) error
  ListByTicket(ctx context.Context, ticketID string) ([]*Comment, error)
}

type KnowledgeRepository interface {
  FindByContext(ctx context.Context, q string) ([]*KnowledgeBaseEntry, error)
  Upsert(ctx context.Context, e *KnowledgeBaseEntry) error
}

// ai ports
type AISuggestionService interface {
  SuggestMitigation(ctx context.Context, description string) (mitigation string, confidence float64, err error)
}

type AITrainingService interface {
  LearnFromResolved(ctx context.Context, ticket *Ticket, comments []*Comment) error
}

// infra port
type NotificationService interface {
  NotifyUser(ctx context.Context, userID string, payload Notification) error
}
```

---

## 5. Use Cases (Application Layer)

Setiap use case harus ringan, idempotent, dan diuji menggunakan table-driven tests.

* **CreateTicketUseCase**

    * Input: CreateTicketDTO (title, description, createBy, optional: createViaAI boolean)
    * Flow: call AISuggestionService (non-blocking or bounded timeout) → persist Ticket with aiInsight if applicable → emit event `TicketCreated` → notify assignees/admins if auto-assign rule match
    * Output: TicketDTO

* **AISuggestUseCase**

    * Input: text
    * Flow: call AISuggestionService → if confidence >= threshold return mitigation; else ask clarifying questions

* **CommentOnTicketUseCase**

    * Persist comment → emit `CommentAdded` → notify relevant parties

* **AssignTicketUseCase**

    * Validate admin role → update assignedTo → append audit entry

* **ResolveTicketUseCase**

    * Admin adds final mitigation → change status to RESOLVED → call AITrainingService asynchronously (but must complete call or enqueue before returning success) → record resolution time

* **GetMetricsUseCase**

    * Query ReportingService → return MetricReportDTO

---

## 6. Eventing & Integration Patterns

* Use domain events for cross-cutting flows: `TicketCreated`, `TicketAssigned`, `TicketResolved`, `CommentAdded`, `KBUpdated`.
* Events are published to simple internal broker (in-memory for MVP) with adapters for Kafka/RabbitMQ in infra.
* AITrainingService should subscribe to `TicketResolved` and `KBUpdated`.

---

## 7. API Contracts (surface)

* REST endpoints (OpenAPI-ready):

    * POST /v1/tickets → create ticket
    * GET /v1/tickets/{id} → ticket detail + comments
    * POST /v1/tickets/{id}/comments → add comment
    * POST /v1/tickets/{id}/assign → assign ticket
    * POST /v1/tickets/{id}/resolve → resolve ticket
    * GET /v1/kb → list KB
    * POST /v1/ai/suggest → return suggestion
    * GET /v1/reports → metric report

All endpoints must carry correlation-id and be secured via SSO/JWT.

---

## 8. Persistence & Schema (MVP)

Suggested tables (Postgres):

* tickets (id pk, title, description, status, category, priority, created_by, assigned_to, ai_insight jsonb, created_at, updated_at)
* comments (id, ticket_id fk, author_id, role, body, created_at)
* knowledge_base (id, context_label, solution_steps text, tags text[], source text, active bool, updated_at)
* metrics_agg (pre-aggregated if needed)
* audit_logs (id, resource_type, resource_id, action, actor, meta jsonb, created_at)

Add indices on ticket.status, ticket.assigned_to, created_at, and knowledge_base.context_label.

---

## 9. Folder Structure (Starter for Go)

```
/cmd/fixora
/internal
  /domain
    ticket.go
    comment.go
    kb.go
  /usecase
    ticket_usecase.go
    ai_usecase.go
  /ports
    repository.go
    ai.go
    notification.go
  /adapter
    /http
      handler_ticket.go
    /persistence
      postgres_ticket_repo.go
    /ai
      openai_adapter.go (mock impl for tests)
    /notification
      slack_adapter.go
  /infra
    events/broker.go
  /test
    fixtures

/migrations
/docs

```

---

## 10. Testing Strategy

* Domain tests: invariants, status transitions, VO equality.
* Use case tests: mock ports for deterministic flows.
* Adapter integration tests: run with test Postgres and local mock AI (use docker-compose for CI).
* E2E: simulate a full flow with in-memory event broker and mock AI.

---

## 11. Security & Privacy

* Tickets may contain PII — store `description` encrypted at rest or redact patterns (msisdn, email) as configurable sanitizer.
* RBAC: employee only CRUD own tickets; admin roles can act on assigned tickets.
* Audit trail mandatory for assignment & status changes.

---

## 12. Definition of Done (DoD) untuk MVP Stories

* Unit tests ≥ 80% coverage for domain & use cases.
* Integration test for repo + API endpoints.
* OpenAPI spec synced with endpoints.
* Basic telemetry (request duration, error count) wired.

---

## 13. Next Steps (Concrete tasks)

1. Generate `api/openapi.yaml` for endpoints listed.
2. Create schema migrations for tables.
3. Implement TicketRepository + in-memory mock + Postgres adapter.
4. Implement AISuggestionService mock adapter and integrate with CreateTicket.
5. Implement HTTP handlers and minimal React admin/employee pages for MVP flows.
6. Write tests for CreateTicketUseCase and AISuggestUseCase.

---

## 14. Notes on Speckit Alignment

* All ports & use cases are written to be testable & mockable.
* Domain-first: domain invariants live in `internal/domain` and have no infra deps.
* TDD-friendly: tests for every use case; event contracts versioned.

---

*End of plan v1.0*
