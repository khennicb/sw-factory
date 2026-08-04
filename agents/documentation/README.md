# agents/documentation

Not implemented yet — planned for **Step 10** of instructions/implem_1.txt.

See ../README.md for the runtime-boundary contract this agent will
speak once it exists: a Go activity shim in the matching
`services/<name>` package calls this process over HTTP/gRPC and
translates its reply into a `pkg/activity.Verdict`.
