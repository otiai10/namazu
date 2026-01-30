# Our Important Values

- Prioritize readability, testability, maintainability, extendability, and elegance for our source code.
- You are the manager and the agent orchestrator. You should never implement anything yourself, but delegate it to subagents and task agents. Break down tasks into smaller parts and build a PDCA cycle.
- Use `AskUserQuestion` tool as much as possible whenever there are any unclear points before starting actual tasks.
- Use `code-simplifier:code-simplifier` plugin to keep our source code always simple and clean.
- Use `frontend-design` skill when we need implement graphical user interface.

# Autonomous Testing (IMPORTANT)

**You (Claude Code) can and should test autonomously.**

See: `./knowledge/decisions/002-autonomous-testing-setup.md`

```bash
# Run E2E tests without user intervention
./scripts/e2e-test.sh
```

This allows you to:
- Start the server with `--test-mode` (auth disabled)
- Execute API tests
- Verify behavior changes
- Clean up automatically

Always run `./scripts/e2e-test.sh` after implementing features or fixing bugs to verify functionality.

# Our knowledge base

Following knowledge should be stored under `./knowledge` folder:

- `./knowledge/specs/` = Specifications and requirements
- `./knowledge/styles/` = Coding conventions and style guides
- `./knowledge/decisions/` = History of decision making (just for log)

# Project Specific Guide

// TODO
