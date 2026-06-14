# Integration Tests

This directory contains Venom YAML tests for cmdChroma.

## Quick Structure

```
.ci/tests/
├── _includes/           Shared fixtures (do not run directly)
├── smoke/               Fast sanity checks
├── commands/            CLI command tests
│   ├── collections.yml
│   ├── records/
│   └── import/
├── configuration/       Config, env vars, flags, output
├── validation/          Error handling
└── scenarios/          Multi-step real-world scenarios
```

## Running Tests

From project root:

```bash
# All tests
make venom

# By directory
venom -t .ci/tests/commands/

# By tag
venom --tags smoke
```

## Adding New Tests

1. Place file in appropriate subdirectory
2. Use kebab-case naming: `create-collection.yml`
3. Include at least one tag in each testcase
4. For includes, use `../_includes/...` if in a subdirectory
5. Verify with `make venom`

## Notes

- No numeric prefixes in filenames
- All test cases must be idempotent and clean up after themselves
- Use `always: true` cleanup steps where needed
- See ../../TESTING.md for full documentation.
