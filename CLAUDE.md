## ⚠️ MANDATORY LOAD

Before any work in this project, read and obey AGENT_DIRECTIVES.md in the project root.

All rules in that file are hard overrides. They govern:
- Pre-work protocol (dead code cleanup, phased execution)
- Code quality (senior dev override, forced verification, type safety)
- Context management (sub-agent swarming, decay awareness, read budget)
- Edit safety (re-read before/after edit, grep-based rename, import hygiene)
- Commit discipline (atomic commits, no broken commits)
- Communication (state plan, report honestly, no hallucinated APIs)

Violation of any rule is a blocking issue.

---

## Project Overrides

> Add project-specific rules below. These extend AGENT_DIRECTIVES.md, never contradict it.
> Delete or modify the placeholder sections as needed.

### Language & Tooling

<!-- Uncomment and fill the relevant block -->

<!-- GO -->
<!--
- Language: Go
- Min version: 1.22+
- Build: go build ./...
- Lint: go vet ./... && staticcheck ./...
- Test: go test ./... -count=1 -short
- Dependency policy: [strict-zero | minimal | standard]
-->

<!-- TYPESCRIPT -->
<!--
- Language: TypeScript
- Runtime: Node.js 22+ / Bun
- Build: npx tsc --noEmit
- Lint: npx eslint . --quiet
- Test: npm test
- Module system: ESM / CJS / dual
-->

<!-- RUST -->
<!--
- Language: Rust
- Edition: 2021
- Build: cargo build
- Lint: cargo clippy -- -D warnings
- Test: cargo test
-->

<!-- PYTHON -->
<!--
- Language: Python
- Min version: 3.11+
- Lint: ruff check . or flake8
- Type check: mypy .
- Test: pytest
-->

<!-- PHP -->
<!--
- Language: PHP
- Min version: 8.2+
- Lint: php -l <files>
- Test: phpunit or manual
-->

### Architecture Notes

<!-- Describe the project's architecture constraints, e.g.: -->
<!-- - Single binary output -->
<!-- - Monorepo structure -->
<!-- - Microservice boundaries -->
<!-- - Specific patterns to follow (CQRS, hexagonal, etc.) -->

### Dependency Policy

<!-- Options: -->
<!-- - ZERO: No external dependencies allowed -->
<!-- - MINIMAL: External deps require explicit justification -->
<!-- - STANDARD: Use well-maintained packages freely -->
<!-- - List any banned or preferred packages -->

### Known Gotchas

<!-- List anything an AI agent would likely get wrong, e.g.: -->
<!-- - "Don't use X library v3, we're pinned to v2 because of Y" -->
<!-- - "The config package has a global singleton, don't create new instances" -->
<!-- - "Tests require Docker running for integration suite" -->
<!-- - "CI uses Node 20, not 22 — don't use 22-only APIs" -->