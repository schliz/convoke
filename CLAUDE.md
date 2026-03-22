# Important Instructions for AI Coding Agents (always follow these)

- You MUST use the context7 tools to get up-to-date information on all libraries and dependencies used. Never rely on your knowledge of these dependencies, it is outdated.
- The `docs/design` folder is a submodule and should be treated as immutable. These design documents may change but this is out of scope for you, they are maintained manually.
- Load the `backlog` Skill on every turn when working on tasks with the "TASK-" prefix or when requring information on milestone, decisions, documentation, etc.

## Common Patterns and Pitfalls

The oob content replace functionality in HTMX cannot replace table rows or semantic HTML objects that the browser wraps or otherwise escapes when fetching a partial update via HTTP.
