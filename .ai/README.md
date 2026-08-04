# .ai/

Machine-readable project knowledge that `services/repository-intelligence`
will index and serve to agents starting in Step 5 (spec §4's "Repository
Intelligence"). Empty scaffold for now — populated when Step 5 lands.

- `docs/` — architecture notes, ADRs, and other reference docs agents should
  ground their work in.
- `conventions/` — coding style/conventions the Implementation and Review
  agents should follow, expressed in whatever form Step 5 decides (plain
  markdown, structured YAML, etc.).

Until Step 5, `services/repository-intelligence.Stub` returns canned data
regardless of what's in this directory — see that package's `interface.go`.
