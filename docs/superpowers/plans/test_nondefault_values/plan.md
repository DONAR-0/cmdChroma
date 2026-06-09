# Plan: Expand Integration Testing for Non-Default Values

## Overview

This plan outlines how to expand integration testing for `cmdChroma` to verify that the CLI works correctly with non-default configuration values (custom hosts, ports, tenants, databases). Currently, tests assume defaults (`localhost:8000`, `default_tenant`, `default_database`), which limits production readiness validation.

## Current State Analysis

### Default Values in Code

The CLI defines defaults in `cmd/chroma/definitions.go`:

| Flag | Default | Env Vars |
|------|---------|----------|
| `--host` | `localhost` | `CHROMA_HOST` |
| `--port` | `8000` | `CHROMA_PORT` |
| `--tenant` | `default_tenant` | `TENANT`, `CHROMA_TENANT` |
| `--database` | `default_database` | `DATABASE`, `CHROMA_DATABASE` |

### Configuration Flow

```
CLI Flags → config.LoadFromCLI() → ChromaConfig → URL construction → ChromaDB Client
```

Key file: `internal/config/config.go`
- `resolvePaths()` builds URL: `http://{host}:{port}`
- No config file support currently exists

## Testing Gaps

### Currently Tested
- Default values work (implicitly)
- Flag overrides work (`-tenant`, `-database`)
- Environment variable overrides work (`TENANT=`, `DATABASE=`)

### NOT Tested (Production Scenarios)

1. **Custom Host (IP/Docker/K8s)**
   - `chroma t --host 192.168.1.100`
   - `chroma t --host chromadb.production.svc.cluster.local`

2. **Custom Port**
   - `chroma t --port 8080`
   - `chroma t --port 443` (if using HTTPS proxy)

3. **Non-Default Tenant/Database Combinations**
   - Production tenant with production database
   - Multiple tenants in same test run

4. **Environment Variable Priority**
   - CLI flag overrides env var
   - Env var overrides default

5. **Configuration Chaining**
   - All four values different from defaults simultaneously

6. **Connection Timeout with Custom Host**
   - Verify timeout works for unreachable hosts

## Implementation Approach

### Phase 1: Test Infrastructure Enhancement

#### 1.1 Use Docker Compose for Isolated Testing

Create a separate test environment that spins up ChromaDB with non-default config:

```yaml
# .ci/docker/docker-compose.testing.yml
services:
  chroma:
    image: chromadb/chroma:latest
    ports:
      - "8099:8000"
    environment:
      - IS_PERSISTENT=FALSE
    # Non-default tenant/database setup
    command: --worker-thread-count 1
```

#### 1.2 Add Test Configuration Helpers

Create a test utility package that manages:
- Starting/stopping test Chroma instances
- Creating test tenants/databases
- Cleaning up after tests

### Phase 2: New Test Cases

Add comprehensive test coverage for non-default values:

#### 2.1 Test File: `17_NonDefaultConfig.yml`

Test custom host and port combinations:

```yaml
testcases:
  - name: Ping with custom host (localhost:8099)
    steps:
      - type: exec
        script: cmdChroma t --host localhost --port 8099
        assertions:
          - result.code ShouldEqual 0
          - result.systemout ShouldContainSubstring "Successfully connected to ChromaDB"

  - name: Ping with IP address
    steps:
      - type: exec
        script: cmdChroma t --host 127.0.0.1
        assertions:
          - result.code ShouldEqual 0

  - name: Ping connection refused (invalid host)
    steps:
      - type: exec
        script: cmdChroma t --host 192.168.255.254 --port 8000
        assertions:
          - result.code ShouldNotEqual 0
          - result.systemout ShouldContainSubstring "connection failed"
```

#### 2.2 Test File: `18_CustomTenantDatabase.yml`

Test non-default tenant/database operations:

```yaml
testcases:
  - name: Create collection in custom tenant/database
    steps:
      - type: exec
        script: |
          cmdChroma create prod_collection \
            --tenant production_tenant \
            --database production_db
        assertions:
          - result.code ShouldEqual 0
          - result.systemout ShouldContainSubstring "'prod_collection' created"

  - name: Verify tenant/database displayed correctly
    steps:
      - type: exec
        script: cmdChroma test --tenant production_tenant --database production_db
        assertions:
          - result.code ShouldEqual 0
          - result.systemout ShouldContainSubstring "production_tenant"
          - result.systemout ShouldContainSubstring "production_db"

  - name: Create collection using env vars
    steps:
      - type: exec
        script: |
          TENANT=env_tenant DATABASE=env_db cmdChroma create env_collection
        assertions:
          - result.code ShouldEqual 0
          - result.systemout ShouldContainSubstring "'env_collection' created"
```

#### 2.3 Test File: `19_FlagEnvPriority.yml`

Verify flag > env var > default priority:

```yaml
testcases:
  - name: Flag overrides env var
    steps:
      - type: exec
        script: TENANT=env_tenant cmdChroma test --tenant flag_tenant
        assertions:
          - result.systemout ShouldContainSubstring "flag_tenant"
          - result.systemout ShouldNotContainSubstring "env_tenant"

  - name: Env var overrides default
    steps:
      - type: exec
        script: TENANT=env_tenant cmdChroma test
        assertions:
          - result.systemout ShouldContainSubstring "env_tenant"
          - result.systemout ShouldNotContainSubstring "default_tenant"

  - name: All custom via env vars
    steps:
      - type: exec
        script: |
          CHROMA_HOST=localhost \
          CHROMA_PORT=8099 \
          TENANT=custom \
          DATABASE=custom \
          cmdChroma test
        assertions:
          - result.code ShouldEqual 0
          - result.systemout ShouldContainSubstring "custom"
```

#### 2.4 Test File: `20_TimeoutWithCustomHost.yml`

Test timeout behavior with unreachable hosts:

```yaml
testcases:
  - name: Connection timeout on unreachable host
    steps:
      - type: exec
        timeout: 10
        script: |
          cmdChroma t \
            --host 192.168.255.254 \
            --timeout 3
        assertions:
          - result.code ShouldNotEqual 0
          - result.systemout ShouldContainSubstring "timeout"

  - name: Quick timeout on localhost port with no listener
    steps:
      - type: exec
        timeout: 15
        script: cmdChroma t --port 8098 --timeout 5
        assertions:
          - result.code ShouldNotEqual 0
```

### Phase 3: Parallel Test Support

Add test cases that can run in parallel with different Chroma instances:

```yaml
# .ci/tests/21_ParallelIsolated.yml
testcases:
  - name: Instance A on port 8097
    parallel: true
    steps:
      - type: exec
        script: cmdChroma t --port 8097
        assertions:
          - result.code ShouldEqual 0

  - name: Instance B on port 8098
    parallel: true
    steps:
      - type: exec
        script: cmdChroma t --port 8098
        assertions:
          - result.code ShouldEqual 0
```

## Test Data Requirements

### For Custom Tenant/Database Tests

ChromaDB's API allows creating tenants and databases dynamically. Tests should:

1. Create test tenant via API: `POST /api/v2/tenants/{tenant}/databases`
2. Verify collection exists in correct tenant/database
3. Clean up after test

Current test fixtures already show this pattern is supported:
- `CHROMA_TENANT=some_tenant` works in existing tests

## Environment Setup

### Option A: Separate Docker Compose

Create a dedicated test environment:

```yaml
# .ci/docker/docker-compose.prod-test.yml
version: '3.8'
services:
  chroma-prod:
    image: chromadb/chroma:latest
    ports:
      - "8099:8000"
    environment:
      - ANONYMIZED_TELEMETRY=false
```

### Option B: Use Existing Chroma with Ports

Use the existing ChromaDB instance but test against different ports/services.

## Acceptance Criteria

1. **Custom Host Tests Pass**
   - Can connect to `localhost:8099`
   - Can connect to `127.0.0.1`
   - Proper error for unreachable hosts

2. **Custom Port Tests Pass**
   - Can connect on non-standard ports
   - Port validation rejects invalid values

3. **Custom Tenant/Database Tests Pass**
   - Collections created in correct tenant/database
   - Env vars work correctly
   - Flag overrides work correctly

4. **Priority Tests Pass**
   - CLI flag > env var > default

5. **Timeout Tests Pass**
   - Timeout works for unreachable hosts
   - Timeout is configurable

## Non-Goals

- Testing Kubernetes ingress/service discovery (out of scope)
- Testing HTTPS/TLS (future enhancement)
- Config file support (separate feature)

## Estimated Effort

| Phase | Tasks | Complexity |
|-------|-------|------------|
| Phase 1 | Docker compose setup, test helpers | Medium |
| Phase 2 | 4 new test files (~50 test cases) | Low-Medium |
| Phase 3 | Parallel test support | Low |

Total estimated: 2-3 days

## References

- Current test structure: `.ci/tests/`
- Venom test framework: https://github.com/ovh/venom
- ChromaDB API: https://docs.trychroma.com/api-reference
- Current defaults: `cmd/chroma/definitions.go`

## Task Tracker

| Task ID | Description | Status | Priority |
|---------|-------------|--------|----------|
| [Task 1](#task-1--docker-compose-setup-for-non-default-testing) | Docker Compose setup for non-default testing | ☐ Not Started | High |
| [Task 2](#task-2--test-17_nondefaultconfigyml) | Create test file 17_NonDefaultConfig.yml | ☐ Not Started | High |
| [Task 3](#task-3--test-18_customtenantdatabaseyml) | Create test file 18_CustomTenantDatabase.yml | ☐ Not Started | High |
| [Task 4](#task-4--test-19_flagenvpriorityyml) | Create test file 19_FlagEnvPriority.yml | ☐ Not Started | High |
| [Task 5](#task-5--test-20_timeoutwithcustomhostyml) | Create test file 20_TimeoutWithCustomHost.yml | ☐ Not Started | Medium |
| [Task 6](#task-6--test-21_parallelisolatedyml) | Create test file 21_ParallelIsolated.yml | ☐ Not Started | Low |
| [Task 7](#task-7--verify-all-tests-pass) | Verify all tests pass locally | ☐ Not Started | High |

---

## Task Details

### Task 1: Docker Compose setup for non-default testing

**Objective:** Create Docker Compose configuration for testing CLI with non-default ChromaDB instances.

**Files to Create:**
- `.ci/docker/docker-compose.testing.yml` - Main test environment
- `.ci/docker/docker-compose.prod-test.yml` - Production-like test environment

**Files to Modify:**
- `.ci/scripts/run-venom.sh` - Add support for custom port testing

**Acceptance Criteria:**
- Docker Compose can start a ChromaDB instance on port 8099
- Existing tests pass with default setup
- New port 8099 can be tested via CLI

**Effort:** 2 hours

---

### Task 2: Test 17_NonDefaultConfig.yml

**Objective:** Test custom host and port combinations.

**File to Create:** `.ci/tests/17_NonDefaultConfig.yml`

**Test Cases:**
| Test Name | Description |
|-----------|-------------|
| Ping with custom port 8099 | `cmdChroma t --port 8099` |
| Ping with IP 127.0.0.1 | `cmdChroma t --host 127.0.0.1` |
| Ping with custom host + port | `cmdChroma t --host localhost --port 8099` |
| Connection refused (invalid host) | `--host 192.168.255.254` should fail |
| Connection refused (invalid port) | `--port 60999` should fail validation |

**Acceptance Criteria:**
- All test cases pass
- Error messages are clear and actionable

**Effort:** 1 hour

---

### Task 3: Test 18_CustomTenantDatabase.yml

**Objective:** Test non-default tenant/database operations.

**File to Create:** `.ci/tests/18_CustomTenantDatabase.yml`

**Test Cases:**
| Test Name | Description |
|-----------|-------------|
| Create collection in custom tenant | `create prod_collection --tenant production_tenant` |
| Verify tenant/database displayed | `test --tenant prod --database prod` shows correct values |
| Create via env vars | `TENANT=x DATABASE=y create coll` |
| Mixed flag/env usage | Flag overrides env var for tenant |

**Acceptance Criteria:**
- Collections are created in correct tenant/database
- Output reflects correct tenant/database values

**Effort:** 1.5 hours

---

### Task 4: Test 19_FlagEnvPriority.yml

**Objective:** Verify CLI flag > env var > default priority.

**File to Create:** `.ci/tests/19_FlagEnvPriority.yml`

**Test Cases:**
| Test Name | Description |
|-----------|-------------|
| Flag overrides env var | `--tenant flag_val` wins over `TENANT=env_val` |
| Env var overrides default | `TENANT=env_val` wins over default_tenant |
| All env vars | All 4 values via environment |
| All flags | All 4 values via CLI flags |
| Mixed env/flags | Host from env, port from flag |

**Acceptance Criteria:**
- Priority order is respected: flag > env > default
- Output confirms which value is active

**Effort:** 1 hour

---

### Task 5: Test 20_TimeoutWithCustomHost.yml

**Objective:** Test timeout behavior with unreachable hosts.

**File to Create:** `.ci/tests/20_TimeoutWithCustomHost.yml`

**Test Cases:**
| Test Name | Description |
|-----------|-------------|
| Timeout on unreachable host | `--host 192.168.255.254 --timeout 3` |
| Timeout on closed port | `--port 60998 --timeout 5` |
| Default timeout works | 30 second default timeout |
| Custom timeout | `--timeout 5` with unreachable host |

**Acceptance Criteria:**
- Timeout is respected
- Error message mentions "timeout" or "deadline"
- No hang after timeout

**Effort:** 1 hour

---

### Task 6: Test 21_ParallelIsolated.yml

**Objective:** Test parallel execution with different Chroma instances.

**File to Create:** `.ci/tests/21_ParallelIsolated.yml`

**Test Cases:**
| Test Name | Description |
|-----------|-------------|
| Instance A on port 8097 | Connect to first test instance |
| Instance B on port 8098 | Connect to second test instance |
| Isolation verification | Each instance is independent |

**Acceptance Criteria:**
- Multiple instances can be tested simultaneously
- No interference between test runs

**Effort:** 0.5 hours

---

### Task 7: Verify all tests pass

**Objective:** Run full test suite and verify all new and existing tests pass.

**Actions:**
1. Run `make test` locally
2. Run venom tests: `venom run`
3. Verify no regressions in existing tests
4. Commit all new test files

**Acceptance Criteria:**
- All tests pass (100% pass rate)
- No new lint warnings
- Test output is clean

**Effort:** 0.5 hours

---

## Summary

| Phase | Tasks | Total Effort |
|-------|-------|---------------|
| Phase 1 (Infrastructure) | Task 1 | 2 hours |
| Phase 2 (Test Cases) | Tasks 2-6 | 5 hours |
| Phase 3 (Verification) | Task 7 | 0.5 hours |
| **Total** | **7 tasks** | **~7.5 hours** |