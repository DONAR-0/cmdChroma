# Testing Guide

This document describes how to run, organize, and add integration tests for cmdChroma.

**For general development guidelines, code style, and documentation standards, see [CONTRIBUTING.md](CONTRIBUTING.md).**

## Test Directory Structure

```
.ci/tests/
├── _includes/                    # Shared fixtures and assertions
│   └── assertions/
├── smoke/                        # Fast, run-on-every-commit tests
│   └── basic-execution.yml
├── commands/                     # CLI command tests by domain
│   ├── collections.yml
│   ├── databases.yml
│   ├── tenant.yml
│   ├── query.yml
│   ├── records/                 # CRUD operations on records
│   │   ├── list.yml
│   │   ├── create.yml
│   │   ├── update.yml
│   │   └── delete.yml
│   ├── import/                  # Data import commands
│   │   ├── jsonl.yml
│   │   └── parquet.yml
├── configuration/               # Configuration-related tests
│   ├── log_level.yml
│   ├── environment_vars.yml
│   ├── config_file.yml
│   ├── config_init.yml
│   └── output_modes.yml
├── validation/                  # Input validation and error messages
│   ├── help_text.yml
│   └── flag_combinations.yml
└── scenarios/                   # Cross-cutting feature scenarios
    ├── custom_tenant_database.yml
    ├── timeout_handling.yml
    ├── parallel_isolated.yml
    ├── auto_create_database.yml
    └── non_default_config.yml
```

## Running Tests

### Run all tests
```bash
make venom
```

### Run tests by directory
```bash
# Only command tests
venom -t .ci/tests/commands/

# Only configuration tests
venom -t .ci/tests/configuration/
```

### Run specific test files by category
```bash
# All command tests
venom -t .ci/tests/commands/

# All configuration tests
venom -t .ci/tests/configuration/

# All smoke tests
venom -t .ci/tests/smoke/
```

**Note:** While test cases include `tags` for documentation purposes, Venom does not support filtering by tags. Use directory-based selection to run test categories.

### Run a specific test file
```bash
venom -t .ci/tests/commands/collections.yml
```

## Adding a New Test

1. **Determine target directory** based on test category:
   - `commands/` for CLI command functionality
   - `configuration/` for config file, flags, env vars, output modes
   - `validation/` for input validation and error handling
   - `scenarios/` for multi-step cross-cutting scenarios
   - `smoke/` for fast sanity checks

2. **Name the file** using kebab-case (lowercase with hyphens):
   - ✅ `create-collection.yml`
   - ❌ `CreateCollection.yml`
   - ❌ `01_CreateCollection.yml`

3. **YAML structure** (minimal):
   ```yaml
   version: "2"
   name: Test Chroma CLI - Brief Description
   import:
     - ../_includes/collection_required.yml   # only if needed
   testcases:
     - name: Test case name
       tags: ["integration"]   # required: at least one tag
       steps:
         - type: exec
           script: cmdChroma --log-level info collname
           assertions:
             - result.code ShouldEqual 0
   ```

4. **Use tags** for categorization:
   - `smoke` - fast, run on every commit
   - `slow` - long-running
   - `requires-server` - needs ChromaDB
   - `jsonl`, `parquet` - format-specific
   - `integration` - general integration tests

5. **Include shared fixtures** when available from `_includes/`:
   - `../_includes/collection_required.yml` ensures collection exists
   - `../_includes/file_required.yml` ensures test file exists

6. **Relative paths**: Use `../` to reach `_includes/` from subdirectories. Example: from `commands/records/list.yml`, include `../_includes/collection_required.yml`.

7. **Run verification**:
   ```bash
   make venom
   ```
   Ensure your test passes and appears in the logs with the correct file path.

## Common Pitfalls

- **Incorrect include paths**: When test file is in a subdirectory, include paths must use `../` to go up to `_includes/`.
- **Missing tags**: All test cases must have at least one tag.
- **Hardcoded numeric prefixes**: Do not use `01_`, `02_`, etc. Use descriptive kebab-case names.
- **Scripts accessing repo**: Use relative paths that work from the test's working directory (usually same directory as the YAML file).

## CI/CD Integration

The CI runs tests using the `make venom` target, which invokes `.ci/scripts/run-venom.sh`. The script:
- Discovers all `.yml` files recursively under `.ci/tests/` excluding `_includes/`
- Sets up ChromaDB instances via Docker Compose
- Runs Venom with verbose logging
- Returns non-zero exit code on any failure

## Debugging Failures

1. Check the `.ci/logs/` directory for detailed logs and dump files.
2. The `venom.log` file contains a full trace.
3. Dump files (`*.dump.json`) capture stdout/stderr for each step.
