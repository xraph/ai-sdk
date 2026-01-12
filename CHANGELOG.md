# Changelog

All notable changes to this project will be documented in this file.

## [v0.0.2] - 2026-01-12

### Initial Release

- 648ba69 (HEAD -> main, origin/main) Update README to reflect project name change from "Forge AI SDK" to "AI SDK"
- a8f18b1 Add ReAct and Plan-Execute agent documentation and examples
- d5ad552 Refactor agent architecture to unify execution control
- 8b9a208 Update Go version to 1.25.5 across CI and release workflows
- e04e197 Update Go module dependencies and version
- 47d23ff Enhance artifact ID generation with concurrency support
- 04fdd75 Add gosec configuration and update security scan commands
- 850298d Refactor test cases and improve error handling in artifact and handoff tests
- 8dde4d7 Update Makefile for AI SDK project
- 77ff378 Refactor whitespace in pgvector.go for improved code readability
- 8df4acc Remove obsolete integration documentation files - These files were no longer relevant as the integration module has reached completion and the documentation has been consolidated into more current formats.
- 6165c72 Update Go module dependencies and improve memory cache implementation
- 01e5d4e Remove trailing whitespace in integration test files for cleaner code formatting
- 0cb7fa0 Implement path sanitization to prevent directory traversal vulnerabilities in audio and multimodal file handling
- 178be92 Update Go module dependencies and clean up whitespace in benchmark and integration files
- 3ca1d8e Improve parsing robustness in score extraction and concurrency handling in tests
- bef838e Update contact information and attribution in README.md to reflect new support email, Discord link, and documentation URL, changing team attribution from Forge Team to XRaph Team.
- d192400 Update installation instructions in README.md to reflect the correct package path for the AI SDK.
- 5c59fe8 Enhance README.md with new Integrations section
- a6f6682 Refactor code formatting and update Go version in module files
- 4ea80cd Add in-memory cache and state store implementations
- c182774 Add foundational components for agent strategies and integrations
- 1fdc6c3 Enhance README.md with new section on Native Guardrails, highlighting zero-cost, low-latency PII and toxicity detection. Update feature comparison table to reflect improvements in cost tracking, guardrails, caching, and provider support. Add architecture philosophy and cost comparison to clarify SDK advantages over competitors.
- 3e281b8 Update README.md to reflect changes in production readiness status, marking the SDK as "Alpha" instead of "Enterprise features" to better represent its current development stage.
- 0d28d7a Enhance README.md with updated framework comparison and key differentiators for Forge AI SDK, Vercel AI SDK, and LangChain. Expand feature set details, including type safety, memory systems, and production readiness, to provide clearer insights into each SDK's strengths and use cases.
- 1ef15b3 Update go-utils dependency to v0.0.5 in go.mod and remove outdated entry from go.sum
- 5f2e29d Refactor tests for artifact and stream functionality
- a2694e5 Add comprehensive tests for workflow functionality

