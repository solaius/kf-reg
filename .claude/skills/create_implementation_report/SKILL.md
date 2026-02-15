# Skill: create_implementation_report

## When to use
Use this skill after completing a milestone or significant implementation task to create a standardized report documenting what was done, why it was done, and how it was done.

## Report location
Reports are written to `catalog_of_catalogs/<phase-folder>/implementation-reports/` using the naming convention:
```
M<number>_<short-slug>.md
```
Example: `M1_plugin-framework-hardening.md`

## Report structure

Every implementation report MUST follow this template:

```markdown
# M<N>: <Title>

**Date**: YYYY-MM-DD
**Status**: Complete
**Phase**: <Phase name>

## Summary
2-3 sentence overview of what this milestone delivered and why it matters.

## Motivation
- Why was this work needed?
- What problem or gap did it address?
- What spec requirements does it satisfy? (reference FR/AC numbers if applicable)

## What Changed

### Files Created
| File | Purpose |
|------|---------|
| `path/to/file.go` | Brief description |

### Files Modified
| File | Change |
|------|--------|
| `path/to/file.go` | Brief description of the change |

## How It Works

### <Component/Concept 1>
Explain the design and implementation. Include code snippets for key interfaces,
structs, or patterns. Keep snippets short (5-15 lines) and focused on the public
contract, not internal details.

### <Component/Concept 2>
Continue for each major component.

## Key Design Decisions
| Decision | Rationale | Alternatives Considered |
|----------|-----------|------------------------|
| Brief decision | Why this approach | What else was evaluated |

## Testing
- What tests were added?
- How to run them?
- What do they cover?

## Verification
Step-by-step commands to verify this milestone works end-to-end:
```bash
# command 1
# command 2
```

## Dependencies & Impact
- What does this milestone enable? (downstream milestones)
- What does it depend on? (upstream work)
- Any backward compatibility notes?

## Open Items
- Any remaining gaps, known limitations, or future improvements?
```

## Guidelines
- Be concise — reports should be scannable in 2-3 minutes
- Focus on the public contract (interfaces, endpoints, config format) over internal details
- Include actual file paths relative to repo root
- Include actual code snippets for key interfaces — readers should understand the API without reading the source
- Reference spec requirements (FR1-FR12, AC1-AC6) where applicable
- List ALL files created or modified — completeness matters for review
- Keep design decision rationale honest — explain tradeoffs, not just the chosen path
- Verification section should have copy-pasteable commands

## Validation
- Report follows the template structure above
- All sections are present (Summary through Open Items)
- File paths are accurate and complete
- Code snippets compile and reflect actual implementation
- Verification commands work
