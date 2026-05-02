# Backend Integration Rules

Backend integration coverage for this skill means HTTP-level verification against the real application wiring under `backend/internal/platform/app/*_integration_test.go`.

## What Counts

- Requests go through the HTTP surface, not direct service or repository calls.
- Tests use the shared database harness so persistence, transactions, and schema behavior are part of the evidence.
- Coverage proves the happy path for the slice's primary API flow.
- Coverage proves authorization or tenant boundary behavior for the same slice.
- Coverage proves invalid transitions or invalid state changes, not just payload validation.
- Coverage proves derived read models after writes when the user experience depends on a follow-up list, dashboard, summary, or other read path.

## Required Expectations

- Put slice-level integration tests in `backend/internal/platform/app/*_integration_test.go`.
- Prefer table-driven coverage when the same endpoint has multiple role or transition cases, but keep each case readable.
- Use the shared DB harness consistently so setup and cleanup match the rest of the app integration suite.
- Assert both transport-level outcomes and persisted state where the scenario requires it.
- For mutation flows, follow the write with the relevant read path when the slice depends on downstream visibility.
- Record missing integration coverage as a gap, not as an implied "frontend will catch it".

## Minimum Scenario Set

- Happy path: valid request, expected HTTP status, expected response body, expected stored state.
- Authorization: unauthenticated, wrong actor, or wrong tenant behavior as applicable.
- Invalid transition: the slice rejects impossible or disallowed state changes with the correct status and no corrupt write.
- Derived read model: the follow-up read reflects the write correctly, or the test explicitly proves why no derived read applies.

## What Does Not Count As Integration Coverage

- Direct unit tests of handlers, services, repositories, or domain objects.
- Mocked HTTP handlers that skip the real app wiring.
- Pure schema or migration tests with no slice-level HTTP behavior.
- Manual curl notes with no committed automated test.
- Frontend e2e tests used as a substitute for missing backend integration coverage.
- A single happy-path test that ignores authz, invalid transitions, or read-model effects required by the slice.
