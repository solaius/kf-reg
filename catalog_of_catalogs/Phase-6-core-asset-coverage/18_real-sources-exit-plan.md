# Phase 6 exit criteria plan: real sources and minimal catalogs

## What "real sources" means for Phase 6
A "real source" should be one of:
- A Git repository (public or internal) containing catalogs as code
- An HTTP endpoint that behaves like a remote catalog service
- An OCI registry repository containing published artifacts

Local YAML files are still allowed as the baseline, but at least two non-YAML provider types should be proven end-to-end.

## Recommended minimal set to hit exit criteria quickly
### Catalog 1: Agents (Git provider)
- Source: Git repo containing agent definitions
- Why: agents are the top priority and Git is the natural workflow

### Catalog 2: Prompt Templates (Git provider)
- Source: same Git repo or separate prompt library repo
- Why: prompts are also natural in Git and immediately visible in UX

### Catalog 3: Policies (Git provider)
- Source: Git repo with OPA bundles and policy metadata
- Why: policy-as-code is proven and realistic

### Catalog 4: Notebooks (Git provider)
- Source: Git repo with notebooks and metadata
- Why: notebooks are commonly stored in Git and validate nbformat easily

Optional Catalog 5: Guardrails (OCI provider)
- Source: OCI registry hosting guardrail bundles
- Why: a strong example of assets as artifacts

Optional Catalog 6: Datasets (HTTP provider)
- Source: remote dataset catalog endpoint or a local mock that follows HTTP provider contract
- Why: demonstrates HTTP provider reuse and pagination

## Data realism guidelines
- Use actual file formats for artifacts:
  - ipynb notebooks
  - rego policy files and bundle structures
  - prompt template files
- Avoid placeholder values that cannot be validated
- Ensure every catalog entry has:
  - license and provenance
  - lifecycle state
  - owner fields
  - at least one filterable tag

## Definition of done
- At least four catalogs above load successfully and are usable in UI and CLI
- At least one of those catalogs uses a provider type beyond YAML (Git recommended for the minimum)
- Action flows validate and apply succeed for those catalogs
