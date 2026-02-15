# Catalog of Catalogs, Plugin-Based AI Asset Catalog (High-Level Spec)

## Executive summary
We are evolving the Kubeflow Model Catalog into a generic, extensible catalog platform that can host many AI asset catalogs within one catalog-server. Models remain fully supported with zero breaking changes. New AI asset types (for example MCP servers, datasets, prompt templates, agents, evaluation benchmarks) are added as independent plugins that share common infrastructure: configuration, database connection, filtering, pagination, and documentation.

This spec describes what we are building, why it exists, and the requirements Claude should satisfy while choosing the most practical implementation approach in the Kubeflow codebase.

## The product in one sentence
A single discovery and browsing experience for all AI building blocks, implemented as a unified server that loads asset-type plugins, each with its own API surface and data providers.

## Why this matters
Teams building AI applications and platforms are managing more than models. The number of assets and their interdependencies are growing, and users need consistent discovery, search, and governance-adjacent metadata across asset types without deploying and learning a new service for each.

We also want a scalable engineering model where adding a new asset type is mostly a schema and provider exercise, not a reinvention of infrastructure.

## Core outcomes
- One catalog-server hosts multiple catalogs, one per asset type
- Model Catalog remains fully compatible for current consumers
- Adding a new asset type is fast and repeatable
- UI and CLI can discover and present new asset types consistently
- Cross-asset references can be expressed and resolved in an extensible way

## What this is not
- Not a deployment controller or orchestration system
- Not the system of record for lifecycle governance of every asset
- Not a replacement for specialized registries, artifact stores, or execution engines

## Reading guide
- 01 covers goals, scope, and non-goals
- 02 defines terminology and the core mental model
- 03 describes user journeys and use cases
- 04 through 09 contain requirements
- 10 through 13 contain validation, rollout, and risk management

