# Skill: plan_and_align

## When to use
Use this skill at the start of work on this project or when scope changes.

## Inputs to read first
- All numbered spec files in this pack
- PROGRAMMING_GUIDELINES.md (repo conventions and required workflow)

## Steps
1. Summarize the project goal and non-goals in 8 to 12 bullets
2. List assumptions and open questions (call out anything that could trigger a breaking change)
3. Propose a milestone plan with concrete outputs and verifiable checkpoints
4. For each milestone, name the contract changes (OpenAPI paths and schemas), the codegen steps, and the tests you will run
5. Identify risks and propose mitigations
6. Confirm the validation loop you will follow until CI is green

## Validation loop
Run this loop whenever you change API or generated code inputs:
- make gen
- make lint
- make test
- make openapi/validate (or the catalog OpenAPI validation target if different)

## Output
- A short plan in markdown
- A checklist of tasks per milestone
- The commands you will run to validate each milestone
