# AGENTS.md

This document provides guidance for AI coding assistants working with the `sesh` project.

## Testing

- Write unit tests for business logic
- Use integration tests for database operations
- Mock external dependencies (tmux, git) where appropriate
- Test files should be in the same package as the code they test
- Rust unit tests MUST be written in subdirectories, not in source code files to avoid cluttering agent context.

## Important Notes for AI Assistants

1. Use anyhow for rust error handling
2. Write high level documentation for all public features in the `docs/` folder as markdown files.
3. Any diagrams in documentation or plans should be in mermaid format.
3. Every crate/internal module must have a README.md file explaining its purpose and usage.
4. internal readmes must be concise and to the point, and link to other related documentation when relevant.
5. When modifying existing code that relates to any existing documentation, ensure that the documentation is updated accordingly.
5. Follow Rust's idiomatic practices and conventions.
6. You must ensure that any changed code is buildable and passes tests before finalizing changes.

