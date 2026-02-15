# CLI Product Spec

## CLI goals

- Provide a fast, scriptable interface for both AI engineers and ops
- Mirror UI workflows so docs and training are consistent
- Produce stable, machine readable output formats

## CLI shape

Command group

- catalog <resource> <action>

Resources

- plugins
- assets
- sources
- status
- refresh
- validate

Actions

- list, get, describe
- create, update, delete
- enable, disable
- trigger, watch

## Required behaviors

Output

- Default output is human readable
- Support json output for automation
- Support table output for quick scanning

Auth and targeting

- Support targeting a specific Kubeflow endpoint
- Support namespace or profile scoping if applicable
- Respect RBAC and return clear authorization errors

Discoverability

- catalog plugins list shows what the server supports and how to address it
- catalog assets list requires plugin selection unless a cross plugin list is explicitly supported

Error handling

- Non zero exit codes on failures
- Structured error output for json mode

## MVP command examples

- catalog plugins list
- catalog plugins get <name>
- catalog assets list --plugin mcp --filter 'name ILIKE "%vector%"'
- catalog assets get --plugin mcp --id <id>
- catalog sources list --plugin model
- catalog sources validate --plugin mcp --file sources-patch.yaml
- catalog sources apply --plugin mcp --file sources-patch.yaml
- catalog refresh trigger --plugin mcp --source internal-servers
- catalog status --plugin mcp

## Acceptance Criteria

- All MVP commands work against a local dev deployment
- All MVP commands have tests for parsing and error behavior
- Help text is complete and examples are included
