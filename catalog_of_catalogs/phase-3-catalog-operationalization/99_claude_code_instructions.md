# How Claude Code Should Execute Phase 3

_Last updated: 2026-02-16_

## How to use these specs
- Read 00 through 12 before coding
- Treat 03, 04, 05, 06, 07, 08 as the “must implement” specs
- Follow repository programming guidelines (see PROGRAMMING_GUIDELINES.md in repo root)

## Guardrails
- Contract-first: update OpenAPI before implementing new endpoints or fields
- Do not manually edit generated code
- Keep backward compatibility for existing model catalog APIs
- Prefer small PRs per milestone with passing tests

## Workflow expectations
For each milestone:
- Implement the feature
- Add or update tests proving it works in real mode
- Update docs or examples as needed
- Verify acceptance criteria with a runnable command sequence

## Completion promise
Do not stop after code compiles
Iterate until:
- acceptance criteria are met
- E2E smoke tests pass
- generated code checks are clean
