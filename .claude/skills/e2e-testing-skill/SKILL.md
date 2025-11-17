---
name: e2e-playwright-testing
description: Comprehensive end-to-end test creation, and management. You MUST activate this skill when the user mentions "e2e", "end-to-end", "playwright", or any work involving the `e2e/` folder.
---

# End-to-End Playwright Test Creation and Management

<purpose>
This skill guides you through a systematic three-phase approach for creating, implementing, and healing end-to-end Playwright tests. All E2E test work MUST use three specialized sub-agents in sequence to ensure comprehensive test coverage and reliability.
</purpose>

<core_mandate>
**CRITICAL**: You MUST use all three sub-agents in the specified order for ANY E2E test work. Never write, modify, or debug E2E tests directly. Always delegate to the appropriate sub-agent.

The three required sub-agents are:
1. **playwright-test-planner** - Creates comprehensive test plans
2. **playwright-test-generator** - Implements automated browser tests
3. **playwright-test-healer** - Debugs and fixes failing tests

You must ALWAYS use ALL 3 agents IN SEQUENCE according to the phases below. The only time you can skip the planner and generator is if the user's task involves EXISTING tests.
</core_mandate>

<instructions>

## Three-Phase Testing Workflow

### Phase 1: Planning (MANDATORY FIRST STEP FOR ANY NEW TESTS)
<phase_1>
**Action**: Invoke the `playwright-test-planner` sub-agent

**Purpose**: Create a comprehensive test plan before any implementation

**Required Information to Provide**:
- Feature or user flow to be tested
- Expected user interactions
- Critical paths and edge cases
- Available infrastructure in `e2e/helper/` folder
- Any relevant context from `e2e/README.md` or `e2e/package.json`

**Output**: A detailed test plan document that will guide implementation
</phase_1>

### Phase 2: Implementation (EXECUTE ONLY AFTER PHASE 1)
<phase_2>
**Action**: Invoke the `playwright-test-generator` sub-agent

**Purpose**: Implement the test plan as executable Playwright test code

**Required Information to Provide**:
- The complete test plan from Phase 1
- Target location in `e2e/` folder for test files
- Relevant helper utilities from `e2e/helper/` (especially testcontainers infrastructure)
- Application URLs, selectors, or API endpoints needed

**Output**: Working Playwright test files in the `e2e/` folder
</phase_2>

### Phase 3: Healing and Validation (EXECUTE IMMEDIATELY AFTER PHASE 2, OR IF YOU ARE ONLY DEALING WITH EXIST TESTS)
<phase_3>
**Action**: Invoke the `playwright-test-healer` sub-agent

**Purpose**: Run tests, identify failures, and automatically fix issues

**Required Information to Provide**:
- Path to the newly created test files
- Test execution results (if already run)
- Access to application logs or error messages

**Expected Behavior**:
- Healer runs the tests
- If failures occur, healer analyzes and attempts fixes
- Repeat healing until tests pass OR maximum reasonable attempts reached

**If Tests Cannot Be Fixed**:
When the healer cannot resolve failures after multiple attempts:
1. Summarize the failure clearly (what's failing, why it's failing)
2. Explain what was attempted and why it didn't work
3. State: "I am unable to remedy this situation automatically. Human intervention is required."
4. Wait for user input before proceeding
</phase_3>

## Iteration and Refinement
<iteration_rules>
After Phase 3 completes:
- If tests achieve 100% pass rate → Success, workflow complete
- If tests have failures and healer couldn't fix → Stop and request human input
- If new issues are discovered → Return to Phase 1 to revise test plan
- Never skip or bypass any phase during iteration
</iteration_rules>

</instructions>

<constraints>

## Absolute Requirements
<absolute_requirements>
1. **No Direct Test Writing**: You MUST NOT write Playwright test code directly. Always use the playwright-test-generator sub-agent.

2. **No Skipped Tests**: Tests that are skipped are unacceptable. If a test is skipped:
   - Analyze the reason (missing infrastructure, environment issues, etc.)
   - Use available infrastructure in `e2e/helper/` (testcontainers, etc.)
   - Add necessary orchestration to create required infrastructure in the test environment
   - Re-attempt the test until it can run

3. **100% Pass Rate Goal**: The target is always 100% passing tests. The only exception is when the healer has exhausted all reasonable attempts and explicitly requests human intervention.

4. **Sequential Phase Execution**: Phases must execute in order:
   - Planning → Implementation → Healing
   - Never skip ahead or work backwards
   - Complete each phase fully before moving to the next

5. **Infrastructure Awareness**: Before invoking sub-agents, always:
   - Check `e2e/helper/` for existing utilities
   - Reference `e2e/README.md` for setup instructions
   - Review `e2e/package.json` for available dependencies and scripts
</absolute_requirements>

## Output Requirements
<output_requirements>
1. **Test Files**: All test files should be created in the appropriate location within the `e2e/` folder
2. **No Separate Reports**: Do NOT create separate report files or summary documents
3. **Inline Summaries**: Provide brief inline summaries of what was accomplished after each phase
4. **Test Results**: Include test execution results showing pass/fail status
</output_requirements>

</constraints>

<examples>

<example>
<scenario>User requests: "Create E2E tests for the user login flow"</scenario>

<correct_approach>
**Phase 1 - Planning**:
```
Invoking playwright-test-planner to create test plan for user login flow...

Providing context:
- User flow: Login page → Enter credentials → Dashboard
- Edge cases: Invalid credentials, password reset, remember me
- Infrastructure: Using testcontainers from e2e/helper/ for database
```

**Phase 2 - Implementation**:
```
Invoking playwright-test-generator with the test plan...

Test plan includes:
- Happy path: valid credentials
- Error path: invalid credentials
- Edge case: password reset link
- Edge case: remember me functionality

Generating tests in: e2e/auth/login.spec.ts
```

**Phase 3 - Healing**:
```
Invoking playwright-test-healer to run and validate tests...

Test Results (Initial):
- ✓ should login with valid credentials
- ✗ should show error with invalid credentials (selector not found)
- ✓ should navigate to password reset
- ✗ should persist session with remember me (timeout)

Healer attempting fixes...
- Fixed: Updated selector for error message
- Fixed: Added wait for session storage

Re-running tests...
All tests passing ✓ (4/4)
```

Summary: Successfully created and validated login flow E2E tests with 100% pass rate.
</correct_approach>
</example>

<example>
<scenario>Test is skipped due to missing test database</scenario>

<correct_approach>
**During Phase 3 - Healing**:
```
Test Results:
- ⊘ should update user profile (SKIPPED - database not available)

Analysis: Test skipped because test database isn't running.

Solution: The e2e/helper/ folder contains testcontainers infrastructure.

Invoking playwright-test-healer with instruction:
"Configure test to use testcontainers to spin up PostgreSQL database before test execution. Reference e2e/helper/database.ts for existing patterns."

Healer adding database orchestration...
- Added beforeAll() hook to start database container
- Added afterAll() hook to cleanup container
- Updated test to use containerized database

Re-running test...
✓ should update user profile

Test now passing with proper infrastructure orchestration.
```
</correct_approach>
</example>

<example>
<scenario>Healer cannot fix tests after multiple attempts</scenario>

<correct_approach>
**After multiple healing cycles**:
```
Test Results (Attempt 3):
- ✗ should process payment (API returns 500 error)

Healer Analysis:
- Test code is correct
- Selectors are valid
- Infrastructure is properly configured
- Issue: Payment service API is returning 500 Internal Server Error
- Root cause appears to be in the application code, not the test

Summary of Attempts:
1. Updated API request format - still failed
2. Added retry logic with backoff - still failed
3. Verified test data and authentication - still failed

Conclusion: The payment service endpoint has an application-level bug that is causing legitimate test failures. The test correctly identifies the issue.

I am unable to remedy this situation automatically. Human intervention is required to:
1. Investigate the payment service API (status 500 error)
2. Fix the underlying application bug
3. Re-run tests after application fix is deployed
```

[Wait for human input]
</correct_approach>
</example>

</examples>

<context>
**Project Structure Context**:
- `e2e/` - Root folder for all end-to-end tests
- `e2e/helper/` - Contains infrastructure utilities (testcontainers, setup helpers)
- `e2e/README.md` - Project-specific testing documentation
- `e2e/package.json` - Test dependencies and scripts

**Sub-Agent Roles**:
- **playwright-test-planner**: Strategic planning, test case identification, coverage analysis
- **playwright-test-generator**: Code implementation, selector creation, test file generation
- **playwright-test-healer**: Execution, debugging, automatic fixing, validation
</context>

<reasoning_guidance>
When working with E2E tests, always think through:

1. **Before Planning Phase**:
   - What is the user trying to test?
   - What flows or features are involved?
   - What infrastructure already exists?

2. **Before Implementation Phase**:
   - Is the test plan comprehensive?
   - Do I have all the information the generator needs?
   - Which helpers from `e2e/helper/` should be used?

3. **During Healing Phase**:
   - Are failures due to test code or application bugs?
   - Can infrastructure orchestration solve skipped tests?
   - Have I exhausted reasonable automatic fixes before requesting human help?

4. **After Completion**:
   - Is the pass rate 100% or explained?
   - Are all tests running (none skipped)?
   - Is the summary clear and concise?
</reasoning_guidance>
