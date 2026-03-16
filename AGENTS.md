# Agent Rules and Guidelines

This document contains all coding standards, conventions and best practices recommended for the TgTaps project.
This is NOT a strict set of rules, but a set of recommendations to help you write better code.

---

## Table of Contents

- [Engineering philosophy](#engineering-philosophy)
- [Backend guidelines](#backend-guidelines)
  - [Code style](#code-style)
  - [Boolean naming](#boolean-naming)
  - [Add reasonable new lines between logical statements](#add-reasonable-new-lines-between-logical-statements)
  - [Comments](#comments)
  - [Controllers](#controllers)
  - [Dependency injection (DI)](#dependency-injection-di)
  - [Migrations](#migrations)
  - [Refactoring](#refactoring)
  - [Testing](#testing)
  - [Time handling](#time-handling)
  - [CRUD examples](#crud-examples)
- [Frontend guidelines](#frontend-guidelines)
  - [React component structure](#react-component-structure)

---

## Engineering philosophy

**Think like a skeptical senior engineer and code reviewer. Don't just do what was asked—also think about what should have been asked.**

⚠️ **Balance vigilance with pragmatism:** Catch real issues, not theoretical ones. Don't let perfect be the enemy of good.

### Task context assessment:

**First, assess the task scope:**

- **Trivial** (typos, formatting, simple field adds): Apply directly with minimal analysis
- **Standard** (CRUD, typical features): Brief assumption check, proceed
- **Complex** (architecture, security, performance-critical): Full analysis required
- **Unclear** (ambiguous requirements): Always clarify assumptions first

### For non-trivial tasks:

1. **Restate the objective and list assumptions** (explicit + implicit)
   - If any assumption is shaky, call it out clearly
   - Distinguish between what's specified and what you're inferring

2. **Propose appropriate solutions:**
   - For complex tasks: 2–3 viable approaches (including a simpler baseline)
   - Recommend one with clear tradeoffs
   - Consider: complexity, maintainability, performance, future extensibility

3. **Identify risks proactively:**
   - Edge cases and boundary conditions
   - Security/privacy pitfalls
   - Performance risks and scalability concerns
   - Operational concerns (deployment, observability, rollback, monitoring)

4. **Handle ambiguity:**
   - If requirements are ambiguous, make a reasonable default and proceed
   - Clearly label your assumptions
   - Document what would change under alternative assumptions

5. **Deliver quality:**
   - Provide a solution that is correct, testable, and maintainable
   - Include minimal tests or validation steps
   - Follow project testing philosophy: prefer controller tests over unit tests
   - Follow all project guidelines from this document

6. **Self-review before finalizing:**
   - Ask: "What could go wrong?"
   - Patch the answer accordingly
   - Verify edge cases are handled

7. **Fix the reason, not the symptom:**
   - If you find a bug or issue, ask "Why did this happen?" and fix the root cause
   - Avoid quick fixes that don't address underlying problems

### Application guidelines:

**Scale your response to the task:**

- **Trivial changes:** Steps 5-6 only (deliver quality + self-review)
- **Standard features:** Steps 1, 5-6 (restate + deliver + review)
- **Complex/risky changes:** All steps 1-6
- **Ambiguous requests:** Steps 1, 4 mandatory

**Be proportionally thorough—brief for simple tasks, comprehensive for risky ones. Avoid analysis paralysis.**

---

## Backend guidelines

### Naming

Variables and functions naming are the most important part of code readability. Always choose descriptive and meaningful names that clearly indicate the purpose and intent of the code.

Avoid abbreviations, unless they are widely accepted and unambiguous (e.g., `ID`, `URL`, `HTTP`). Use consistent naming conventions across the codebase.

Do not use one-two letters. For example:

Bad:

```
    u := users.getUser()

	pr, pw := io.Pipe()

    r := bufio.NewReader(pr)
```

Good:

```
    user := users.GetUser()

    pipeReader, pipeWriter := io.Pipe()

    bufferedReader := bufio.NewReader(pipeReader)
```

Exclusion: widely used variables like "db", "ctx", "req", "res", etc.

### Code style

**Always place private methods to the bottom of file**

This rule applies to ALL Go files including tests, services, controllers, repositories, etc.

In Go, exported (public) functions/methods start with uppercase letters, while unexported (private) ones start with lowercase letters.

#### Structure order:

1. Type definitions and constants
2. Public methods/functions (uppercase)
3. Private methods/functions (lowercase)

#### Examples:

**Service with methods:**

```go
type UserService struct {
    repository *UserRepository
}

// Public methods first
func (s *UserService) CreateUser(user *User) error {
    if err := s.validateUser(user); err != nil {
        return err
    }
    return s.repository.Save(user)
}

func (s *UserService) GetUser(id uuid.UUID) (*User, error) {
    return s.repository.FindByID(id)
}

// Private methods at the bottom
func (s *UserService) validateUser(user *User) error {
    if user.Name == "" {
        return errors.New("name is required")
    }
    return nil
}
```

**Package-level functions:**

```go
package utils

// Public functions first
func ProcessData(data []byte) (Result, error) {
    cleaned := sanitizeInput(data)
    return parseData(cleaned)
}

func ValidateInput(input string) bool {
    return isValidFormat(input) && checkLength(input)
}

// Private functions at the bottom
func sanitizeInput(data []byte) []byte {
    // implementation
}

func parseData(data []byte) (Result, error) {
    // implementation
}

func isValidFormat(input string) bool {
    // implementation
}

func checkLength(input string) bool {
    // implementation
}
```

**Test files:**

```go
package user_test

// Public test functions first
func Test_CreateUser_ValidInput_UserCreated(t *testing.T) {
    user := createTestUser()
    result, err := service.CreateUser(user)

    assert.NoError(t, err)
    assert.NotNil(t, result)
}

func Test_GetUser_ExistingUser_ReturnsUser(t *testing.T) {
    user := createTestUser()
    // test implementation
}

// Private helper functions at the bottom
func createTestUser() *User {
    return &User{
        Name:  "Test User",
        Email: "test@example.com",
    }
}

func setupTestDatabase() *Database {
    // setup implementation
}
```

**Controller example:**

```go
type ProjectController struct {
    service *ProjectService
}

// Public HTTP handlers first
func (c *ProjectController) CreateProject(ctx *gin.Context) {
    var request CreateProjectRequest
    if err := ctx.ShouldBindJSON(&request); err != nil {
        c.handleError(ctx, err)
        return
    }
    // handler logic
}

func (c *ProjectController) GetProject(ctx *gin.Context) {
    projectID := c.extractProjectID(ctx)
    // handler logic
}

// Private helper methods at the bottom
func (c *ProjectController) handleError(ctx *gin.Context, err error) {
    ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
}

func (c *ProjectController) extractProjectID(ctx *gin.Context) uuid.UUID {
    return uuid.MustParse(ctx.Param("projectId"))
}
```

#### Key points:

- **Exported/Public** = starts with uppercase letter (CreateUser, GetProject)
- **Unexported/Private** = starts with lowercase letter (validateUser, handleError)
- This improves code readability by showing the public API first
- Private helpers are implementation details, so they go at the bottom
- Apply this rule consistently across ALL Go files in the project

---

### Boolean naming

**Always prefix boolean variables with verbs like `is`, `has`, `was`, `should`, `can`, etc.**

This makes the code more readable and clearly indicates that the variable represents a true/false state.

#### Good examples:

```go
type User struct {
    IsActive    bool
    IsVerified  bool
    HasAccess   bool
    WasNotified bool
}

type BackupConfig struct {
    IsEnabled       bool
    ShouldCompress  bool
    CanRetry        bool
}

// Variables
isInProgress := true
wasCompleted := false
hasPermission := checkPermissions()
```

#### Bad examples:

```go
type User struct {
    Active    bool  // Should be: IsActive
    Verified  bool  // Should be: IsVerified
    Access    bool  // Should be: HasAccess
}

type BackupConfig struct {
    Enabled   bool  // Should be: IsEnabled
    Compress  bool  // Should be: ShouldCompress
    Retry     bool  // Should be: CanRetry
}

// Variables
inProgress := true   // Should be: isInProgress
completed := false   // Should be: wasCompleted
permission := true   // Should be: hasPermission
```

#### Common boolean prefixes:

- **is** - current state (IsActive, IsValid, IsEnabled)
- **has** - possession or presence (HasAccess, HasPermission, HasError)
- **was** - past state (WasCompleted, WasNotified, WasDeleted)
- **should** - intention or recommendation (ShouldRetry, ShouldCompress)
- **can** - capability or permission (CanRetry, CanDelete, CanEdit)
- **will** - future state (WillExpire, WillRetry)

---

### Add reasonable new lines between logical statements

**Add blank lines between logical blocks to improve code readability.**

Separate different logical operations within a function with blank lines. This makes the code flow clearer and helps identify distinct steps in the logic.

#### Guidelines:

- Add blank line before final `return` statement
- Add blank line after variable declarations before using them
- Add blank line between error handling and subsequent logic
- Add blank line between different logical operations

#### Bad example (without spacing):

```go
func (t *Task) BeforeSave(tx *gorm.DB) error {
	if len(t.Messages) > 0 {
		messagesBytes, err := json.Marshal(t.Messages)
		if err != nil {
			return err
		}
		t.MessagesJSON = string(messagesBytes)
	}
	return nil
}

func (t *Task) AfterFind(tx *gorm.DB) error {
	if t.MessagesJSON != "" {
		var messages []onewin_dto.TaskCompletionMessage
		if err := json.Unmarshal([]byte(t.MessagesJSON), &messages); err != nil {
			return err
		}
		t.Messages = messages
	}
	return nil
}
```

#### Good example (with proper spacing):

```go
func (t *Task) BeforeSave(tx *gorm.DB) error {
	if len(t.Messages) > 0 {
		messagesBytes, err := json.Marshal(t.Messages)
		if err != nil {
			return err
		}

		t.MessagesJSON = string(messagesBytes)
	}

	return nil
}

func (t *Task) AfterFind(tx *gorm.DB) error {
	if t.MessagesJSON != "" {
		var messages []onewin_dto.TaskCompletionMessage
		if err := json.Unmarshal([]byte(t.MessagesJSON), &messages); err != nil {
			return err
		}

		t.Messages = messages
	}

	return nil
}
```

#### More examples:

**Service method with multiple operations:**

```go
func (s *UserService) CreateUser(request *CreateUserRequest) (*User, error) {
	// Validate input
	if err := s.validateUserRequest(request); err != nil {
		return nil, err
	}

	// Create user entity
	user := &User{
		ID:    uuid.New(),
		Name:  request.Name,
		Email: request.Email,
	}

	// Save to database
	if err := s.repository.Create(user); err != nil {
		return nil, err
	}

	// Send notification
	s.notificationService.SendWelcomeEmail(user.Email)

	return user, nil
}
```

**Repository method with query building:**

```go
func (r *Repository) GetFiltered(filters *Filters) ([]*Entity, error) {
	query := storage.GetDb().Model(&Entity{})

	if filters.Status != "" {
		query = query.Where("status = ?", filters.Status)
	}

	if filters.CreatedAfter != nil {
		query = query.Where("created_at > ?", filters.CreatedAfter)
	}

	var entities []*Entity
	if err := query.Find(&entities).Error; err != nil {
		return nil, err
	}

	return entities, nil
}
```

**Repository method with error handling:**

Bad (without spacing):

```go
func (r *Repository) FindById(id uuid.UUID) (*models.Task, error) {
	var task models.Task
	result := storage.GetDb().Where("id = ?", id).First(&task)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}
		return nil, result.Error
	}
	return &task, nil
}
```

Good (with proper spacing):

```go
func (r *Repository) FindById(id uuid.UUID) (*models.Task, error) {
	var task models.Task

	result := storage.GetDb().Where("id = ?", id).First(&task)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("task not found")
		}

		return nil, result.Error
	}

	return &task, nil
}
```

---

### Comments

#### Guidelines

1. **No obvious comments** - Don't state what the code already clearly shows
2. **Functions and variables should have meaningful names** - Code should be self-documenting
3. **Comments for unclear code only** - Only add comments when code logic isn't immediately clear

#### Key principles:

- **Code should tell a story** - Use descriptive variable and function names
- **Comments explain WHY, not WHAT** - The code shows what happens, comments explain business logic or complex decisions
- **Prefer refactoring over commenting** - If code needs explaining, consider making it clearer instead
- **API documentation is required** - Swagger comments for all HTTP endpoints are mandatory
- **Complex algorithms deserve comments** - Mathematical formulas, business rules, or non-obvious optimizations
- **Do not write summary sections in .md files unless directly requested** - Avoid adding "Summary" or "Conclusion" sections at the end of documentation files unless the user explicitly asks for them

#### Example of useless comments:

**1. Obvious SQL comment:**

```sql
// Create projects table
CREATE TABLE projects (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
```

**2. Obvious function call comment:**

```go
// Create test project
project := CreateTestProject(projectName, user, router)
```

**3. Redundant function comment:**

```go
// CreateValidLogItems creates valid log items for testing
func CreateValidLogItems(count int, uniqueID string) []logs_receiving.LogItemRequestDTO {
```

---

### Controllers

#### Controller guidelines:

1. **When we write controller:**
   - We combine all routes to single controller
   - Names them as `.WhatWeDo` (not "handlers") concept

2. **We use gin and `*gin.Context` for all routes**

   Example:

   ```go
   func (c *TasksController) GetAvailableTasks(ctx *gin.Context) ...
   ```

3. **We document all routes with Swagger in the following format:**

```go
package audit_logs

import (
    "net/http"

    user_models "databasus-backend/internal/features/users/models"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type AuditLogController struct {
    auditLogService *AuditLogService
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
    // All audit log endpoints require authentication (handled in main.go)
    auditRoutes := router.Group("/audit-logs")

    auditRoutes.GET("/global", c.GetGlobalAuditLogs)
    auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)
}

// GetGlobalAuditLogs
// @Summary Get global audit logs (ADMIN only)
// @Description Retrieve all audit logs across the system
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/global [get]
func (c *AuditLogController) GetGlobalAuditLogs(ctx *gin.Context) {
    user, isOk := ctx.MustGet("user").(*user_models.User)
    if !isOk {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
        return
    }

    request := &GetAuditLogsRequest{}
    if err := ctx.ShouldBindQuery(request); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    response, err := c.auditLogService.GetGlobalAuditLogs(user, request)
    if err != nil {
        if err.Error() == "only administrators can view global audit logs" {
            ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
        return
    }

    ctx.JSON(http.StatusOK, response)
}

// GetUserAuditLogs
// @Summary Get user audit logs
// @Description Retrieve audit logs for a specific user
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User ID"
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/users/{userId} [get]
func (c *AuditLogController) GetUserAuditLogs(ctx *gin.Context) {
    user, isOk := ctx.MustGet("user").(*user_models.User)
    if !isOk {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
        return
    }

    userIDStr := ctx.Param("userId")
    targetUserID, err := uuid.Parse(userIDStr)
    if err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    request := &GetAuditLogsRequest{}
    if err := ctx.ShouldBindQuery(request); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    response, err := c.auditLogService.GetUserAuditLogs(targetUserID, user, request)
    if err != nil {
        if err.Error() == "insufficient permissions to view user audit logs" {
            ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
        return
    }

    ctx.JSON(http.StatusOK, response)
}
```

---

### Dependency injection (DI)

For DI files use **implicit fields declaration styles** (especially for controllers, services, repositories, use cases, etc., not simple data structures).

#### Instead of:

```go
var orderController = &OrderController{
    orderService:   orderService,
    botUserService: bot_users.GetBotUserService(),
    botService:     bots.GetBotService(),
    userService:    users.GetUserService(),
}
```

#### Use:

```go
var orderController = &OrderController{
    orderService,
    bot_users.GetBotUserService(),
    bots.GetBotService(),
    users.GetUserService(),
}
```

**This is needed to avoid forgetting to update DI style when we add new dependency.**

#### Force such usage

Please force such usage if file look like this (see some services\controllers\repos definitions and getters):

```go
var orderBackgroundService = &OrderBackgroundService{
    orderService:           orderService,
    orderPaymentRepository: orderPaymentRepository,
    botService:             bots.GetBotService(),
    paymentSettingsService: payment_settings.GetPaymentSettingsService(),

    orderSubscriptionListeners: []OrderSubscriptionListener{},
}

var orderController = &OrderController{
    orderService:   orderService,
    botUserService: bot_users.GetBotUserService(),
    botService:     bots.GetBotService(),
    userService:    users.GetUserService(),
}

func GetUniquePaymentRepository() *repositories.UniquePaymentRepository {
    return uniquePaymentRepository
}

func GetOrderPaymentRepository() *repositories.OrderPaymentRepository {
    return orderPaymentRepository
}

func GetOrderService() *OrderService {
    return orderService
}

func GetOrderController() *OrderController {
    return orderController
}

func GetOrderBackgroundService() *OrderBackgroundService {
    return orderBackgroundService
}

func GetOrderRepository() *repositories.OrderRepository {
    return orderRepository
}
```

#### SetupDependencies() pattern

**All `SetupDependencies()` functions must use sync.Once to ensure idempotent execution.**

This pattern allows `SetupDependencies()` to be safely called multiple times (especially in tests) while ensuring the actual setup logic executes only once.

**Implementation pattern:**

```go
package feature

import (
    "sync"
    "sync/atomic"
    "databasus-backend/internal/util/logger"
)

var (
    setupOnce sync.Once
    isSetup   atomic.Bool
)

func SetupDependencies() {
    wasAlreadySetup := isSetup.Load()

    setupOnce.Do(func() {
        // Initialize dependencies here
        someService.SetDependency(otherService)
        anotherService.AddListener(listener)

        isSetup.Store(true)
    })

    if wasAlreadySetup {
        logger.GetLogger().Warn("SetupDependencies called multiple times, ignoring subsequent call")
    }
}
```

**Why this pattern:**

- **Tests can call multiple times**: Test setup often calls `SetupDependencies()` multiple times without issues
- **Thread-safe**: Works correctly with concurrent calls (nanoseconds or seconds apart)
- **Idempotent**: Subsequent calls are safe, only log warning
- **No panics**: Does not break tests or production code on multiple calls

**Key Points:**

1. Check `isSetup.Load()` **before** calling `Do()` to detect previous executions
2. Set `isSetup.Store(true)` **inside** the `Do()` closure after setup completes
3. Log warning if already setup (helps identify unnecessary duplicate calls)
4. All setup logic must be inside the `Do()` closure

---

### Background services

**All background service `Run()` methods must panic if called multiple times to prevent corrupted states.**

Background services run infinite loops and must never be started twice on the same instance. Multiple calls indicate a serious bug that would cause duplicate goroutines, resource leaks, and data corruption.

**Implementation pattern:**

```go
package feature

import (
    "context"
    "fmt"
    "sync"
    "sync/atomic"
)

type BackgroundService struct {
    // ... existing fields ...
    runOnce sync.Once
    hasRun  atomic.Bool
}

func (s *BackgroundService) Run(ctx context.Context) {
    wasAlreadyRun := s.hasRun.Load()

    s.runOnce.Do(func() {
        s.hasRun.Store(true)

        // Existing infinite loop logic
        ticker := time.NewTicker(1 * time.Minute)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                s.doWork()
            }
        }
    })

    if wasAlreadyRun {
        panic(fmt.Sprintf("%T.Run() called multiple times", s))
    }
}
```

**Why panic instead of warning:**

- **Prevents corruption**: Multiple `Run()` calls would create duplicate goroutines consuming resources
- **Fails fast**: Catches critical bugs immediately in tests and production
- **Clear indication**: Panic clearly indicates a serious programming error
- **Applies everywhere**: Same protection in tests and production

**When this applies:**

- All background services with infinite loops
- Registry services (BackupNodesRegistry, RestoreNodesRegistry)
- Scheduler services (BackupsScheduler, RestoresScheduler)
- Worker nodes (BackuperNode, RestorerNode)
- Cleanup services (AuditLogBackgroundService, DownloadTokenBackgroundService)

**Key Points:**

1. Check `hasRun.Load()` **before** calling `Do()` to detect previous executions
2. Set `hasRun.Store(true)` **inside** the `Do()` closure before starting work
3. **Always panic** if already run (never just log warning)
4. All run logic must be inside the `Do()` closure
5. This pattern is **thread-safe** for any timing (concurrent or sequential calls)

---

### Migrations

When writing migrations:

- Write them for PostgreSQL
- For PRIMARY UUID keys use `gen_random_uuid()`
- For time use `TIMESTAMPTZ` (timestamp with zone)
- Split table, constraint and indexes declaration (table first, then other one by one)
- Format SQL in pretty way (add spaces, align columns types), constraints split by lines

#### Example:

```sql
CREATE TABLE marketplace_info (
    bot_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title             TEXT NOT NULL,
    description       TEXT NOT NULL,
    short_description TEXT NOT NULL,
    tutorial_url      TEXT,
    info_order        BIGINT NOT NULL DEFAULT 0,
    is_published      BOOLEAN NOT NULL DEFAULT FALSE
);

ALTER TABLE marketplace_info_images
    ADD CONSTRAINT fk_marketplace_info_images_bot_id
    FOREIGN KEY (bot_id)
    REFERENCES marketplace_info (bot_id);
```

---

### Refactoring

When applying changes, **do not forget to refactor old code**.

You can shortify, make more readable, improve code quality, etc. Common logic can be extracted to functions, constants, files, etc.

**After each large change with more than ~50-100 lines of code:**

- Always run `make lint` (from backend root folder)
- If you change frontend, run `npm run format` (from frontend root folder)

---

### Testing

**After writing tests, always launch them and verify that they pass.**

#### Test naming format

Use these naming patterns:

- `Test_WhatWeDo_WhatWeExpect`
- `Test_WhatWeDo_WhichConditions_WhatWeExpect`

#### Examples from real codebase:

- `Test_CreateApiKey_WhenUserIsProjectOwner_ApiKeyCreated`
- `Test_UpdateProject_WhenUserIsProjectAdmin_ProjectUpdated`
- `Test_DeleteApiKey_WhenUserIsProjectMember_ReturnsForbidden`
- `Test_GetProjectAuditLogs_WithDifferentUserRoles_EnforcesPermissionsCorrectly`
- `Test_ProjectLifecycleE2E_CompletesSuccessfully`

#### Testing philosophy

**Prefer controllers over unit tests:**

- Test through HTTP endpoints via controllers whenever possible
- Avoid testing repositories, services in isolation - test via API instead
- Only use unit tests for complex model logic when no API exists
- Name test files `controller_test.go` or `service_test.go`, not `integration_test.go`

**Extract common logic to testing utilities:**

- Create `testing.go` or `testing/testing.go` files for shared test utilities
- Extract router creation, user setup, models creation helpers (in API, not just structs creation)
- Reuse common patterns across different test files

**Refactor existing tests:**

- When working with existing tests, always look for opportunities to refactor and improve
- Extract repetitive setup code to common utilities
- Simplify complex tests by breaking them into smaller, focused tests
- Replace inline test data creation with reusable helper functions
- Consolidate similar test patterns across different test files
- Make tests more readable and maintainable for other developers

**Clean up test data:**

- If the feature supports cleanup operations (DELETE endpoints, cleanup methods), use them in tests
- Clean up resources after test execution to avoid test data pollution
- Use `defer` statements or explicit cleanup calls at the end of tests
- Prioritize using API methods for cleanup (not direct database deletion)
- Examples:
  - CRUD features: delete created records via DELETE endpoint
  - File uploads: remove uploaded files
  - Background jobs: stop schedulers or cancel running tasks
- Skip cleanup only when:
  - Tests run in isolated transactions that auto-rollback
  - Cleanup endpoint doesn't exist yet
  - Test explicitly validates failure scenarios where cleanup isn't possible

**Example:**

```go
func Test_BackupLifecycle_CreateAndDelete(t *testing.T) {
    router := createTestRouter()
    workspace := workspaces_testing.CreateTestWorkspace("Test", owner)

    // Create backup config
    config := createBackupConfig(t, router, workspace.ID, owner.Token)

    // Cleanup at end of test
    defer deleteBackupConfig(t, router, workspace.ID, config.ID, owner.Token)

    // Test operations...
    triggerBackup(t, router, workspace.ID, config.ID, owner.Token)

    // Verify backup was created
    backups := getBackups(t, router, workspace.ID, owner.Token)
    assert.NotEmpty(t, backups)
}
```

#### Testing utilities structure

**Create `testing.go` or `testing/testing.go` files with common utilities:**

```go
package projects_testing

// CreateTestRouter creates unified router for all controllers
func CreateTestRouter(controllers ...ControllerInterface) *gin.Engine {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    v1 := router.Group("/api/v1")
    protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))

    for _, controller := range controllers {
        if routerGroup, ok := protected.(*gin.RouterGroup); ok {
            controller.RegisterRoutes(routerGroup)
        }
    }
    return router
}

// CreateTestProjectViaAPI creates project through HTTP API
func CreateTestProjectViaAPI(name string, owner *users_dto.SignInResponseDTO, router *gin.Engine) (*projects_models.Project, string) {
    request := projects_dto.CreateProjectRequestDTO{Name: name}
    w := MakeAPIRequest(router, "POST", "/api/v1/projects", "Bearer "+owner.Token, request)
    // Handle response...
    return project, owner.Token
}

// AddMemberToProject adds member via API call
func AddMemberToProject(project *projects_models.Project, member *users_dto.SignInResponseDTO, role users_enums.ProjectRole, ownerToken string, router *gin.Engine) {
    // Implementation...
}
```

#### Controller test examples

**Permission-based testing:**

```go
func Test_CreateApiKey_WhenUserIsProjectOwner_ApiKeyCreated(t *testing.T) {
    router := CreateApiKeyTestRouter(GetProjectController(), GetMembershipController())
    owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
    project, _ := projects_testing.CreateTestProjectViaAPI("Test Project", owner, router)

    request := CreateApiKeyRequestDTO{Name: "Test API Key"}
    var response ApiKey
    test_utils.MakePostRequestAndUnmarshal(t, router, "/api/v1/projects/api-keys/"+project.ID.String(), "Bearer "+owner.Token, request, http.StatusOK, &response)

    assert.Equal(t, "Test API Key", response.Name)
    assert.NotEmpty(t, response.Token)
}
```

**Cross-project security testing:**

```go
func Test_UpdateApiKey_WithApiKeyFromDifferentProject_ReturnsBadRequest(t *testing.T) {
    router := CreateApiKeyTestRouter(GetProjectController(), GetMembershipController())
    owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
    owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
    project1, _ := projects_testing.CreateTestProjectViaAPI("Project 1", owner1, router)
    project2, _ := projects_testing.CreateTestProjectViaAPI("Project 2", owner2, router)

    apiKey := CreateTestApiKey("Cross Project Key", project1.ID, owner1.Token, router)

    // Try to update via different project endpoint
    request := UpdateApiKeyRequestDTO{Name: &"Hacked Key"}
    resp := test_utils.MakePutRequest(t, router, "/api/v1/projects/api-keys/"+project2.ID.String()+"/"+apiKey.ID.String(), "Bearer "+owner2.Token, request, http.StatusBadRequest)

    assert.Contains(t, string(resp.Body), "API key does not belong to this project")
}
```

**E2E lifecycle testing:**

```go
func Test_ProjectLifecycleE2E_CompletesSuccessfully(t *testing.T) {
    router := projects_testing.CreateTestRouter(GetProjectController(), GetMembershipController())

    // 1. Create project
    owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
    project := projects_testing.CreateTestProject("E2E Project", owner, router)

    // 2. Add member
    member := users_testing.CreateTestUser(users_enums.UserRoleMember)
    projects_testing.AddMemberToProject(project, member, users_enums.ProjectRoleMember, owner.Token, router)

    // 3. Promote to admin
    projects_testing.ChangeMemberRole(project, member.UserID, users_enums.ProjectRoleAdmin, owner.Token, router)

    // 4. Transfer ownership
    projects_testing.TransferProjectOwnership(project, member.UserID, owner.Token, router)

    // 5. Verify new owner can manage project
    finalProject := projects_testing.GetProject(project.ID, member.Token, router)
    assert.Equal(t, project.ID, finalProject.ID)
}
```

---

### Time handling

**Always use `time.Now().UTC()` instead of `time.Now()`**

This ensures consistent timezone handling across the application.

---

### CRUD examples

This is an example of complete CRUD implementation structure:

#### controller.go

```go
package audit_logs

import (
    "net/http"

    user_models "databasus-backend/internal/features/users/models"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

type AuditLogController struct {
    auditLogService *AuditLogService
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
    // All audit log endpoints require authentication (handled in main.go)
    auditRoutes := router.Group("/audit-logs")

    auditRoutes.GET("/global", c.GetGlobalAuditLogs)
    auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)
}

// GetGlobalAuditLogs
// @Summary Get global audit logs (ADMIN only)
// @Description Retrieve all audit logs across the system
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/global [get]
func (c *AuditLogController) GetGlobalAuditLogs(ctx *gin.Context) {
    user, isOk := ctx.MustGet("user").(*user_models.User)
    if !isOk {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
        return
    }

    request := &GetAuditLogsRequest{}
    if err := ctx.ShouldBindQuery(request); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    response, err := c.auditLogService.GetGlobalAuditLogs(user, request)
    if err != nil {
        if err.Error() == "only administrators can view global audit logs" {
            ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
        return
    }

    ctx.JSON(http.StatusOK, response)
}

// GetUserAuditLogs
// @Summary Get user audit logs
// @Description Retrieve audit logs for a specific user
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User ID"
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/users/{userId} [get]
func (c *AuditLogController) GetUserAuditLogs(ctx *gin.Context) {
    user, isOk := ctx.MustGet("user").(*user_models.User)
    if !isOk {
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
        return
    }

    userIDStr := ctx.Param("userId")
    targetUserID, err := uuid.Parse(userIDStr)
    if err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
        return
    }

    request := &GetAuditLogsRequest{}
    if err := ctx.ShouldBindQuery(request); err != nil {
        ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    response, err := c.auditLogService.GetUserAuditLogs(targetUserID, user, request)
    if err != nil {
        if err.Error() == "insufficient permissions to view user audit logs" {
            ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
            return
        }
        ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
        return
    }

    ctx.JSON(http.StatusOK, response)
}
```

#### controller_test.go

```go
package audit_logs

import (
    "fmt"
    "net/http"
    "testing"
    "time"

    user_enums "databasus-backend/internal/features/users/enums"
    users_middleware "databasus-backend/internal/features/users/middleware"
    users_services "databasus-backend/internal/features/users/services"
    users_testing "databasus-backend/internal/features/users/testing"
    "databasus-backend/internal/storage"
    test_utils "databasus-backend/internal/util/testing"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
)

func Test_GetGlobalAuditLogs_AdminSucceedsAndMemberGetsForbidden(t *testing.T) {
    adminUser := users_testing.CreateTestUser(user_enums.UserRoleAdmin)
    memberUser := users_testing.CreateTestUser(user_enums.UserRoleMember)
    router := createRouter()
    service := GetAuditLogService()
    projectID := uuid.New()

    // Create test logs
    createAuditLog(service, "Test log with user", &adminUser.UserID, nil)
    createAuditLog(service, "Test log with project", nil, &projectID)
    createAuditLog(service, "Test log standalone", nil, nil)

    // Test ADMIN can access global logs
    var response GetAuditLogsResponse
    test_utils.MakeGetRequestAndUnmarshal(t, router,
        "/api/v1/audit-logs/global?limit=10", "Bearer "+adminUser.Token, http.StatusOK, &response)

    assert.GreaterOrEqual(t, len(response.AuditLogs), 3)
    assert.GreaterOrEqual(t, response.Total, int64(3))

    messages := extractMessages(response.AuditLogs)
    assert.Contains(t, messages, "Test log with user")
    assert.Contains(t, messages, "Test log with project")
    assert.Contains(t, messages, "Test log standalone")

    // Test MEMBER cannot access global logs
    resp := test_utils.MakeGetRequest(t, router, "/api/v1/audit-logs/global",
        "Bearer "+memberUser.Token, http.StatusForbidden)
    assert.Contains(t, string(resp.Body), "only administrators can view global audit logs")
}

func Test_GetUserAuditLogs_PermissionsEnforcedCorrectly(t *testing.T) {
    adminUser := users_testing.CreateTestUser(user_enums.UserRoleAdmin)
    user1 := users_testing.CreateTestUser(user_enums.UserRoleMember)
    user2 := users_testing.CreateTestUser(user_enums.UserRoleMember)
    router := createRouter()
    service := GetAuditLogService()
    projectID := uuid.New()

    // Create test logs for different users
    createAuditLog(service, "Test log user1 first", &user1.UserID, nil)
    createAuditLog(service, "Test log user1 second", &user1.UserID, &projectID)
    createAuditLog(service, "Test log user2 first", &user2.UserID, nil)
    createAuditLog(service, "Test log user2 second", &user2.UserID, &projectID)
    createAuditLog(service, "Test project log", nil, &projectID)

    // Test ADMIN can view any user's logs
    var user1Response GetAuditLogsResponse
    test_utils.MakeGetRequestAndUnmarshal(t, router,
        fmt.Sprintf("/api/v1/audit-logs/users/%s?limit=10", user1.UserID.String()),
        "Bearer "+adminUser.Token, http.StatusOK, &user1Response)

    assert.Equal(t, 2, len(user1Response.AuditLogs))
    messages := extractMessages(user1Response.AuditLogs)
    assert.Contains(t, messages, "Test log user1 first")
    assert.Contains(t, messages, "Test log user1 second")

    // Test user can view own logs
    var ownLogsResponse GetAuditLogsResponse
    test_utils.MakeGetRequestAndUnmarshal(t, router,
        fmt.Sprintf("/api/v1/audit-logs/users/%s", user2.UserID.String()),
        "Bearer "+user2.Token, http.StatusOK, &ownLogsResponse)
    assert.Equal(t, 2, len(ownLogsResponse.AuditLogs))

    // Test user cannot view other user's logs
    resp := test_utils.MakeGetRequest(t, router,
        fmt.Sprintf("/api/v1/audit-logs/users/%s", user1.UserID.String()),
        "Bearer "+user2.Token, http.StatusForbidden)

    assert.Contains(t, string(resp.Body), "insufficient permissions")
}

func Test_FilterAuditLogsByTime_ReturnsOnlyLogsBeforeDate(t *testing.T) {
    adminUser := users_testing.CreateTestUser(user_enums.UserRoleAdmin)
    router := createRouter()
    service := GetAuditLogService()
    db := storage.GetDb()
    baseTime := time.Now().UTC()

    // Create logs with different timestamps
    createTimedLog(db, &adminUser.UserID, "Test old log", baseTime.Add(-2*time.Hour))
    createTimedLog(db, &adminUser.UserID, "Test recent log", baseTime.Add(-30*time.Minute))
    createAuditLog(service, "Test current log", &adminUser.UserID, nil)

    // Test filtering - get logs before 1 hour ago
    beforeTime := baseTime.Add(-1 * time.Hour)
    var filteredResponse GetAuditLogsResponse
    test_utils.MakeGetRequestAndUnmarshal(t, router,
        fmt.Sprintf("/api/v1/audit-logs/global?beforeDate=%s", beforeTime.Format(time.RFC3339)),
        "Bearer "+adminUser.Token, http.StatusOK, &filteredResponse)

    // Verify only old log is returned
    messages := extractMessages(filteredResponse.AuditLogs)
    assert.Contains(t, messages, "Test old log")
    assert.NotContains(t, messages, "Test recent log")
    assert.NotContains(t, messages, "Test current log")

    // Test without filter - should get all logs
    var allResponse GetAuditLogsResponse
    test_utils.MakeGetRequestAndUnmarshal(t, router, "/api/v1/audit-logs/global",
        "Bearer "+adminUser.Token, http.StatusOK, &allResponse)
    assert.GreaterOrEqual(t, len(allResponse.AuditLogs), 3)
}

func createRouter() *gin.Engine {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    SetupDependencies()

    v1 := router.Group("/api/v1")
    protected := v1.Group("").Use(users_middleware.AuthMiddleware(users_services.GetUserService()))
    GetAuditLogController().RegisterRoutes(protected.(*gin.RouterGroup))

    return router
}
```

#### di.go

```go
package audit_logs

import (
    users_services "databasus-backend/internal/features/users/services"
    "databasus-backend/internal/util/logger"
)

var auditLogRepository = &AuditLogRepository{}
var auditLogService = &AuditLogService{
    auditLogRepository: auditLogRepository,
    logger:             logger.GetLogger(),
}
var auditLogController = &AuditLogController{
    auditLogService: auditLogService,
}

func GetAuditLogService() *AuditLogService {
    return auditLogService
}

func GetAuditLogController() *AuditLogController {
    return auditLogController
}

func SetupDependencies() {
    users_services.GetUserService().SetAuditLogWriter(auditLogService)
    users_services.GetSettingsService().SetAuditLogWriter(auditLogService)
    users_services.GetManagementService().SetAuditLogWriter(auditLogService)
}
```

#### dto.go

```go
package audit_logs

import "time"

type GetAuditLogsRequest struct {
    Limit      int        `form:"limit"      json:"limit"`
    Offset     int        `form:"offset"     json:"offset"`
    BeforeDate *time.Time `form:"beforeDate" json:"beforeDate"`
}

type GetAuditLogsResponse struct {
    AuditLogs []*AuditLog `json:"auditLogs"`
    Total     int64       `json:"total"`
    Limit     int         `json:"limit"`
    Offset    int         `json:"offset"`
}
```

#### models.go

```go
package audit_logs

import (
    "time"

    "github.com/google/uuid"
)

type AuditLog struct {
    ID        uuid.UUID  `json:"id"        gorm:"column:id"`
    UserID    *uuid.UUID `json:"userId"    gorm:"column:user_id"`
    ProjectID *uuid.UUID `json:"projectId" gorm:"column:project_id"`
    Message   string     `json:"message"   gorm:"column:message"`
    CreatedAt time.Time  `json:"createdAt" gorm:"column:created_at"`
}

func (AuditLog) TableName() string {
    return "audit_logs"
}
```

#### repository.go

```go
package audit_logs

import (
    "databasus-backend/internal/storage"
    "time"

    "github.com/google/uuid"
)

type AuditLogRepository struct{}

func (r *AuditLogRepository) Create(auditLog *AuditLog) error {
    if auditLog.ID == uuid.Nil {
        auditLog.ID = uuid.New()
    }

    return storage.GetDb().Create(auditLog).Error
}

func (r *AuditLogRepository) GetGlobal(limit, offset int, beforeDate *time.Time) ([]*AuditLog, error) {
    var auditLogs []*AuditLog

    query := storage.GetDb().Order("created_at DESC")

    if beforeDate != nil {
        query = query.Where("created_at < ?", *beforeDate)
    }

    err := query.
        Limit(limit).
        Offset(offset).
        Find(&auditLogs).Error

    return auditLogs, err
}

func (r *AuditLogRepository) GetByUser(
    userID uuid.UUID,
    limit, offset int,
    beforeDate *time.Time,
) ([]*AuditLog, error) {
    var auditLogs []*AuditLog

    query := storage.GetDb().
        Where("user_id = ?", userID).
        Order("created_at DESC")

    if beforeDate != nil {
        query = query.Where("created_at < ?", *beforeDate)
    }

    err := query.
        Limit(limit).
        Offset(offset).
        Find(&auditLogs).Error

    return auditLogs, err
}

func (r *AuditLogRepository) GetByProject(
    projectID uuid.UUID,
    limit, offset int,
    beforeDate *time.Time,
) ([]*AuditLog, error) {
    var auditLogs []*AuditLog

    query := storage.GetDb().
        Where("project_id = ?", projectID).
        Order("created_at DESC")

    if beforeDate != nil {
        query = query.Where("created_at < ?", *beforeDate)
    }

    err := query.
        Limit(limit).
        Offset(offset).
        Find(&auditLogs).Error

    return auditLogs, err
}

func (r *AuditLogRepository) CountGlobal(beforeDate *time.Time) (int64, error) {
    var count int64
    query := storage.GetDb().Model(&AuditLog{})

    if beforeDate != nil {
        query = query.Where("created_at < ?", *beforeDate)
    }

    err := query.Count(&count).Error
    return count, err
}
```

#### service.go

```go
package audit_logs

import (
    "errors"
    "log/slog"
    "time"

    user_enums "databasus-backend/internal/features/users/enums"
    user_models "databasus-backend/internal/features/users/models"

    "github.com/google/uuid"
)

type AuditLogService struct {
    auditLogRepository *AuditLogRepository
    logger             *slog.Logger
}

func (s *AuditLogService) WriteAuditLog(
    message string,
    userID *uuid.UUID,
    projectID *uuid.UUID,
) {
    auditLog := &AuditLog{
        UserID:    userID,
        ProjectID: projectID,
        Message:   message,
        CreatedAt: time.Now().UTC(),
    }

    err := s.auditLogRepository.Create(auditLog)
    if err != nil {
        s.logger.Error("failed to create audit log", "error", err)
        return
    }
}

func (s *AuditLogService) CreateAuditLog(auditLog *AuditLog) error {
    return s.auditLogRepository.Create(auditLog)
}

func (s *AuditLogService) GetGlobalAuditLogs(
    user *user_models.User,
    request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
    if user.Role != user_enums.UserRoleAdmin {
        return nil, errors.New("only administrators can view global audit logs")
    }

    limit := request.Limit
    if limit <= 0 || limit > 1000 {
        limit = 100
    }

    offset := max(request.Offset, 0)

    auditLogs, err := s.auditLogRepository.GetGlobal(limit, offset, request.BeforeDate)
    if err != nil {
        return nil, err
    }

    total, err := s.auditLogRepository.CountGlobal(request.BeforeDate)
    if err != nil {
        return nil, err
    }

    return &GetAuditLogsResponse{
        AuditLogs: auditLogs,
        Total:     total,
        Limit:     limit,
        Offset:    offset,
    }, nil
}

func (s *AuditLogService) GetUserAuditLogs(
    targetUserID uuid.UUID,
    user *user_models.User,
    request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
    // Users can view their own logs, ADMIN can view any user's logs
    if user.Role != user_enums.UserRoleAdmin && user.ID != targetUserID {
        return nil, errors.New("insufficient permissions to view user audit logs")
    }

    limit := request.Limit
    if limit <= 0 || limit > 1000 {
        limit = 100
    }

    offset := max(request.Offset, 0)

    auditLogs, err := s.auditLogRepository.GetByUser(targetUserID, limit, offset, request.BeforeDate)
    if err != nil {
        return nil, err
    }

    return &GetAuditLogsResponse{
        AuditLogs: auditLogs,
        Total:     int64(len(auditLogs)),
        Limit:     limit,
        Offset:    offset,
    }, nil
}

func (s *AuditLogService) GetProjectAuditLogs(
    projectID uuid.UUID,
    request *GetAuditLogsRequest,
) (*GetAuditLogsResponse, error) {
    limit := request.Limit
    if limit <= 0 || limit > 1000 {
        limit = 100
    }

    offset := max(request.Offset, 0)

    auditLogs, err := s.auditLogRepository.GetByProject(projectID, limit, offset, request.BeforeDate)
    if err != nil {
        return nil, err
    }

    return &GetAuditLogsResponse{
        AuditLogs: auditLogs,
        Total:     int64(len(auditLogs)),
        Limit:     limit,
        Offset:    offset,
    }, nil
}
```

#### service_test.go

```go
package audit_logs

import (
    "testing"
    "time"

    user_enums "databasus-backend/internal/features/users/enums"
    users_testing "databasus-backend/internal/features/users/testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "gorm.io/gorm"
)

func Test_AuditLogs_ProjectSpecificLogs(t *testing.T) {
    service := GetAuditLogService()
    user1 := users_testing.CreateTestUser(user_enums.UserRoleMember)
    user2 := users_testing.CreateTestUser(user_enums.UserRoleMember)
    project1ID, project2ID := uuid.New(), uuid.New()

    // Create test logs for projects
    createAuditLog(service, "Test project1 log first", &user1.UserID, &project1ID)
    createAuditLog(service, "Test project1 log second", &user2.UserID, &project1ID)
    createAuditLog(service, "Test project2 log first", &user1.UserID, &project2ID)
    createAuditLog(service, "Test project2 log second", &user2.UserID, &project2ID)
    createAuditLog(service, "Test no project log", &user1.UserID, nil)

    request := &GetAuditLogsRequest{Limit: 10, Offset: 0}

    // Test project 1 logs
    project1Response, err := service.GetProjectAuditLogs(project1ID, request)
    assert.NoError(t, err)
    assert.Equal(t, 2, len(project1Response.AuditLogs))

    messages := extractMessages(project1Response.AuditLogs)
    assert.Contains(t, messages, "Test project1 log first")
    assert.Contains(t, messages, "Test project1 log second")
    for _, log := range project1Response.AuditLogs {
        assert.Equal(t, &project1ID, log.ProjectID)
    }

    // Test project 2 logs
    project2Response, err := service.GetProjectAuditLogs(project2ID, request)
    assert.NoError(t, err)
    assert.Equal(t, 2, len(project2Response.AuditLogs))

    messages2 := extractMessages(project2Response.AuditLogs)
    assert.Contains(t, messages2, "Test project2 log first")
    assert.Contains(t, messages2, "Test project2 log second")

    // Test pagination
    limitedResponse, err := service.GetProjectAuditLogs(project1ID,
        &GetAuditLogsRequest{Limit: 1, Offset: 0})
    assert.NoError(t, err)
    assert.Equal(t, 1, len(limitedResponse.AuditLogs))
    assert.Equal(t, 1, limitedResponse.Limit)

    // Test beforeDate filter
    beforeTime := time.Now().UTC().Add(-1 * time.Minute)
    filteredResponse, err := service.GetProjectAuditLogs(project1ID,
        &GetAuditLogsRequest{Limit: 10, BeforeDate: &beforeTime})
    assert.NoError(t, err)
    for _, log := range filteredResponse.AuditLogs {
        assert.True(t, log.CreatedAt.Before(beforeTime))
    }
}

func createAuditLog(service *AuditLogService, message string, userID, projectID *uuid.UUID) {
    service.WriteAuditLog(message, userID, projectID)
}

func extractMessages(logs []*AuditLog) []string {
    messages := make([]string, len(logs))
    for i, log := range logs {
        messages[i] = log.Message
    }
    return messages
}

func createTimedLog(db *gorm.DB, userID *uuid.UUID, message string, createdAt time.Time) {
    log := &AuditLog{
        ID:        uuid.New(),
        UserID:    userID,
        Message:   message,
        CreatedAt: createdAt,
    }
    db.Create(log)
}
```

---

## Frontend guidelines

### React component structure

Write React components with the following structure:

```typescript
interface Props {
   someValue: SomeValue;
}

const someHelperFunction = () => {
    ...
}

export const ReactComponent = ({ someValue }: Props): JSX.Element => {
    // First put states
    const [someState, setSomeState] = useState<...>(...)

    // Then place functions
    const loadSomeData = async () => {
        ...
    }

    // Then hooks
    useEffect(() => {
       loadSomeData();
    });

    // Then calculated values
    const calculatedValue = someValue.calculate();

    return <div> ... </div>
}
```

#### Structure order:

1. **Props interface** - Define component props
2. **Helper functions** (outside component) - Pure utility functions
3. **Component declaration**
   - **States** - `useState` declarations
   - **Functions** - Event handlers and async operations
   - **Hooks** - `useEffect`, `useMemo`, `useCallback`, etc.
   - **Calculated values** - Derived data from props/state
   - **Return** - JSX markup

---

## Summary

This document consolidates all project rules and guidelines. These standards should be followed consistently across all code in the Postgresus project to maintain code quality, readability, and maintainability.
