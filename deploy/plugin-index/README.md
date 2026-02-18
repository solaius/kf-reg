# Supported Plugin Index

This directory contains the index entries for supported catalog plugins.

## Format

Each plugin entry is a YAML file in `plugins/<name>.yaml` following the PluginIndexEntry schema defined in `schema.yaml`.

## Requirements for Supported Plugins

1. Conformance suite passes (latest released server version in range)
2. Compatibility metadata is correct
3. Ownership is declared (named owning team)
4. Security checks pass (license, vuln scan)
5. Documentation kit is complete

## Verification

Run governance checks against any plugin directory:

```bash
catalog-gen validate --governance ./catalog/plugins/<name>
```

## Adding a New Plugin

1. Run the governance checks to confirm the plugin meets requirements
2. Create a new `plugins/<name>.yaml` following the schema
3. Submit a PR to this repository
