# Appendix: Repository patterns relevant to this work

This appendix is a quick extraction of important patterns from PROGRAMMING_GUIDELINES.md so you do not have to hunt for them while implementing.

## Contract-first OpenAPI
- OpenAPI 3.0.3 is used for API definitions
- Specs are merged by scripts in the repository
- Generated server stubs are derived from the spec

## Filtering and pagination
- filterQuery exists as a shared list query mechanism
- Pagination uses pageSize and nextPageToken
- Ordering uses orderBy and sortOrder

## Database and metadata
- Flexible metadata is represented via typed values and property rows where applicable
- Favor additive evolution without frequent schema migrations for every new property

## Testing
- Unit tests and integration tests are expected for new capabilities
- Keep CI checks for generated code and spec sync passing

