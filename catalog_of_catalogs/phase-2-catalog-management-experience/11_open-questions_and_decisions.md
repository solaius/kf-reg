# Open Questions and Decisions

## 1. Source configuration model

Question

- Are sources managed as an API backed resource, or as a file only config with UI and CLI generating patches

Decision needed

- Pick a primary model
- Ensure the alternative is still supported for deployments that cannot allow API writes

## 2. Desired state versus imperative refresh

Question

- Should refresh be on a schedule, on change, on demand, or all three

Decision needed

- Define default behavior and an override mechanism

## 3. Entity lifecycle and mutations

Question

- Are entities ever created or edited directly via API, or is the catalog strictly an ingestion and discovery layer

Decision needed

- Define Phase 2 boundary
- If direct mutation is out of scope, define how users manage source data instead

## 4. Plugin UI hints contract

Question

- What minimal hints are needed for good generic rendering

Decision needed

- Define a small schema for presentation hints that avoids UI coupling

## 5. Cross plugin search

Question

- Do we need a single search across all plugins in Phase 2

Decision needed

- If yes, define ranking and identity semantics
- If no, ensure UX makes plugin switching painless

## 6. RBAC model

Question

- Where to enforce roles and how to map platform identities to viewer versus operator

Decision needed

- Server side enforcement rules
- UI and CLI behavior under partial permissions
