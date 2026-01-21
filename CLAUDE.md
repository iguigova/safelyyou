# SafelyYou Project Instructions

## Learning First

This is a learning exercise. When explaining anything:
- Teach concepts, don't just use them
- Explain the "why" behind decisions
- Define technical terms when first introduced
- Include real-world analogies when helpful

## Design Decisions

Track all design decisions in the "Design Decisions" section of `IMPLEMENTATION_LOG.md`:
- Document alternatives that were considered
- Explain the tradeoffs of each option
- State which option was chosen and why
- This supports defending the implementation in presentation

## Code Quality

After making changes to Go code, always run the linter:
```bash
golangci-lint run
```
Fix any issues before considering the change complete.

## Implementation Log

After each response, update `IMPLEMENTATION_LOG.md` with:
- New concepts explained (with definitions for future reference)
- Commands used and their purpose
- Progress made
- Decisions and their reasoning
- Any new questions or next steps identified
