# agents/test-analysis

Not implemented yet — planned for **Step 8** of instructions/implem_1.txt.

See ../README.md for the runtime-boundary contract this agent will
speak once it exists: a Go activity shim in the matching
`services/<name>` package calls this process over HTTP/gRPC and
translates its reply into a `pkg/activity.Verdict`.

Verdict contract (already reserved in Step 4's Transition Router,
see `pkg/activity/activity.go`): `Fixable | Unfixable | Escalate`.
