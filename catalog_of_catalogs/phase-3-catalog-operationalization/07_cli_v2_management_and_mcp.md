# CLI v2: Management and MCP Catalog Commands

_Last updated: 2026-02-16_

## Goals
Provide a CLI that can:
- manage plugins and sources
- validate and apply source configs
- inspect diagnostics
- browse MCP servers

## Command groups
- `catalog plugins`
  - `list`
  - `get <plugin>`
- `catalog sources`
  - `list <plugin>`
  - `get <plugin> <sourceId>`
  - `validate <plugin> -f source.yaml`
  - `apply <plugin> -f source.yaml`
  - `enable <plugin> <sourceId>`
  - `disable <plugin> <sourceId>`
  - `delete <plugin> <sourceId>`
  - `refresh <plugin> [--source <sourceId>]`
  - `diagnostics <plugin> [--source <sourceId>]`
- `catalog mcp`
  - `list` (with filter flags)
  - `get <name>`
  - `search <query>`

## Global flags
- `--server`
- `--role` (viewer | operator) for local testing and dev
- `--output` (table | json | yaml)

## Output requirements
- Tables are readable and stable
- JSON matches OpenAPI models exactly

## Definition of Done
- CLI works against a real server in E2E setup
- Supports validate, apply, enable, disable, delete, refresh, diagnostics
- MCP list and get work against real MCP catalog data

## Acceptance Criteria
- AC1: `catalog plugins list` returns model and mcp plugins
- AC2: `catalog sources apply` persists and survives restart
- AC3: `catalog sources diagnostics` shows real ingestion errors
- AC4: `catalog mcp list` returns the expected MCP servers and filters correctly
