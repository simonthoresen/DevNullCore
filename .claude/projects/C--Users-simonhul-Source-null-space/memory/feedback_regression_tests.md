---
name: Always create regression tests for bugs
description: When a bug is found and fixed, always create a unit test that would have caught it. This prevents regressions and builds confidence in the test suite.
type: feedback
---

When we find and resolve a bug, always create a unit test that reproduces the issue before the fix and passes after.

**Why:** The slog feedback loop bug (render debug logs → console → re-render → infinite loop) was not caught by the test suite because we only tested controls in isolation, not the message flow between components. The bug caused 70K log lines and CPU spin.

**How to apply:** After fixing any bug, before committing, add a test case in the relevant `_test.go` file that exercises the specific scenario that was broken. For integration bugs that span multiple components, create an integration test that wires the components together with mocks.
