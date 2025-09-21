# Rush

Rush is a fast, lightweight HTTP router for Go with named parameters, wildcards, customizable handlers, and flexible middleware.

## Table of Contents
- [Features](#features)
- [Performance Benchmarks](#performance-benchmarks)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Full Example](#full-example)
- [API Reference](#api-reference)
- [Middleware System](#middleware-system)
- [Route Matching & Precedence](#route-matching--precedence)
- [Custom Error & Default Handlers](#custom-error--default-handlers)
- [Trailing Slash Behavior](#trailing-slash-behavior)
- [Common Patterns](#common-patterns)
- [Important Notes](#important-notes)

## Features

- **Named Parameters**: Extract values from URLs (e.g., `/user/{id}`)
- **Wildcards**: Catch-all routes (e.g., `/static/*` for file serving)
- **Route Precedence**: Automatic most-specific route matching
- **Middleware Groups**: Apply middleware to specific route groups 
- **Prefix Groups**: Group routes with common path prefixes (e.g., `/api/v1`)
- **Customizable handlers**: for `404 Not Found`, `405 Method Not Allowed`, and `OPTIONS`
- **Automatic handling**: of `OPTIONS` and `HEAD` requests
- **Standard Library Compatible**: Works with any `http.Handler` or `http.HandlerFunc`, and standard Go middleware
- **Lightweight & Dependency-Free**: ~300 LOC, zero dependencies, easy-to-read codebase

## Performance Benchmarks

Rush delivers excellent performance, competitive with the fastest Go routers:

| Router      | Static Route | Parameter Route | Memory (Static) | Memory (Param) |
|-------------|--------------|-----------------|-----------------|----------------|
| httprouter  | 1,175 ns/op  | 1,291 ns/op     | 440 B/op        | 504 B/op       |
| **Rush**    | **1,193 ns/op** | **1,387 ns/op** | **440 B/op**   | **440 B/op**   |
| stdmux      | 1,390 ns/op  | 1,849 ns/op     | 440 B/op        | 456 B/op       |
| chi         | 1,635 ns/op  | 2,129 ns/op     | 808 B/op        | 1,145 B/op     |
| gorilla/mux | 2,407 ns/op  | 3,658 ns/op     | 1,289 B/op      | 1,593 B/op     |

*Benchmarks run on Linux AMD with Go 1.24.4 (September 2025). Results may vary by environment.*

## Installation

Install Rush with:

```bash
go get github.com/0xrinful/rush@latest
```

## Quick Start

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/0xrinful/rush"
)

func main() {
    r := rush.New()
    // Register global middleware
    r.Use(loggingMiddleware)

    // Basic route
    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("Hello, Rush!"))
    })

    // Route with URL parameter: /hello/123 → "Hello, user 123"
    r.Get("/hello/{id}", func(w http.ResponseWriter, r *http.Request) {
        id := r.PathValue("id")
        w.Write([]byte("Hello, user " + id))
    })

    // Start server
    http.ListenAndServe(":8080", r)
}

// Example logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Println(r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}
```

## Full Example

```go
package main

import (
    "net/http"
    "github.com/0xrinful/rush"
)

func main() {
    r := rush.New()

    // Custom handlers for common HTTP errors (optional)
    r.NotFound = custom404Handler
    r.MethodNotAllowed = custom405Handler
    r.AutoOptions = customAutoOptionsHandler

    // Redirect requests with a trailing slash to the normalized route
    // Useful for SEO and consistent URLs; only affects existing routes
    r.RedirectTrailingSlash = true 

    // Global middleware - wraps the entire router (affects ALL routes and error handlers)
    r.Use(loggingMiddleware)    // outermost layer (runs first)
    r.Use(corsMiddleware)       // middle layer (runs second)
    r.Use(authMiddleware)       // innermost layer (runs third)

    // Basic routes
    r.Get("/", homeHandler)
    r.Get("/users", listUsersHandler)
    r.Post("/users", createUserHandler)

    // Routes with multiple HTTP methods
    r.HandleFunc("/users", userHandler, "GET", "PUT", "DELETE")

    // Named parameters - access via r.PathValue("id") in handler
    r.Get("/users/{id}/profile", func(w http.ResponseWriter, r *http.Request) {
        userID := r.PathValue("id") // Extract the parameter
        w.Write([]byte("User ID: " + userID))
    })

    // Wildcard routes (only at the end of pattern)
    r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))), "GET")

   // Route grouping with order-dependent middleware
    r.Group(func(r *rush.Router) {
        // Only applies to routes in this group
        r.Use(adminMiddleware)    // Applies to routes defined AFTER this line
        r.Get("/admin/public", publicAdminHandler)  // Uses: adminMiddleware
        
        r.Use(superAdminMiddleware)  // Additional middleware
        r.Get("/admin/private", privateAdminHandler)  // Uses: adminMiddleware + superAdminMiddleware
        r.Delete("/admin/danger", dangerHandler)      // Uses: adminMiddleware + superAdminMiddleware
        
        // Nested groups supported
        r.Group(func(r *rush.Router) {
            // Inherits all middlewares from the parent group
            r.Use(auditMiddleware)  // Additional middleware for this nested group
            r.Delete("/admin/system/reset", systemResetHandler)  // Uses: adminMiddleware + superAdminMiddleware + auditMiddleware
        })
    }) 

    // Prefix grouping for API versioning
    r.GroupWithPrefix("/api/v1", func(r *rush.Router) {
         r.Use(apiMiddleware)
         r.Get("/users/{id}", apiGetUserHandler)     // Full route: /api/v1/users/{id}
         r.Post("/users", apiCreateUserHandler)      // Full route: /api/v1/users
     })

    // Single-route middleware with .With()
    r.With(cacheMiddleware).Get("/cached", cachedHandler)  // Only this route gets cacheMiddleware

    // Start HTTP server
    http.ListenAndServe(":8080", r)
}

// Example custom error handler
func custom404Handler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Custom 404: Page not found"))
}

// Example handlers...
func homeHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Welcome to Rush Router!"))
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Implementation...
}
```

## API Reference

### Router Methods
- `Get(pattern, handler)` - Register GET route
- `Post(pattern, handler)` - Register POST route  
- `Put(pattern, handler)` - Register PUT route
- `Delete(pattern, handler)` - Register DELETE route
- `Patch(pattern, handler)` - Register PATCH route
- `Head(pattern, handler)` - Register HEAD route
- `Options(pattern, handler)` - Register OPTIONS route
- `Handle(pattern, handler, methods...)` - Register route with custom HTTP methods
- `HandleFunc(pattern, handlerFunc, methods...)` - Register HandlerFunc with custom HTTP methods

### Middleware Methods
- `Use(middleware...)` - Register global or group-level middleware
- `With(middleware...)` - Create a new router instance with additional middleware for single routes
- `Group(func(*Router))` - Create a route group with shared middleware
- `GroupWithPrefix(prefix, func(*Router))` - Create a route group with shared path prefix

### Configuration Properties
- `NotFound http.Handler` - Custom 404 Not Found handler
- `MethodNotAllowed http.Handler` - Custom 405 Method Not Allowed handler
- `AutoOptions http.Handler` - Global OPTIONS request handler (used when no specific OPTIONS route is registered)
- `RedirectTrailingSlash bool` - Enable trailing slash redirects

## Middleware System

Rush provides both global and scoped middleware, making it easy to control behavior across your entire app or just for specific groups of routes.

### Root Level (Global Middleware)

Middleware registered at the root level with `r.Use()` wraps the **entire router**, affecting:
- All routes
- Error handlers (404, 405, OPTIONS)
- Everything that goes through the router

```go
r := rush.New()

r.Use(loggingMiddleware)    // Outermost - runs first
r.Use(corsMiddleware)       // Middle layer
r.Use(authMiddleware)       // Innermost - runs last before handler

// Execution order: loggingMiddleware → corsMiddleware → authMiddleware → handler → authMiddleware → corsMiddleware → loggingMiddleware
```

**Key characteristics:**
- Applied to **every request** that hits the router
- Wraps the entire routing logic
- Affects error handlers (404, 405, etc.)
- Cannot be bypassed by any route

### Group Level (Scoped Middleware)

Group-level middleware provides fine-grained control with **order-dependent behavior**:

```go
r.Group(func(r *rush.Router) {
    r.Use(middleware1)           // Applied to routes defined AFTER this line
    
    r.Get("/route1", handler1)   // Gets: middleware1
    
    r.Use(middleware2)           // Additional middleware
    
    r.Get("/route2", handler2)   // Gets: middleware1 + middleware2
    r.Post("/route3", handler3)  // Gets: middleware1 + middleware2
})
```

**Important rules:**
- Routes inherit middlewares defined **above** them in the group
- Routes do **NOT** get middlewares defined **below** them
- Groups inherit their parent group's middleware chain (but not global ones, since they already wrap everything)

### Nested Groups and Inheritance

Groups can be nested, and each level inherits from its parent:

```go
r.Group(func(gr1 *rush.Router) {
    gr1.Use(middleware1)
    
    gr1.Get("/level1", handler1)  // Gets: middleware1
    
    gr1.Group(func(gr2 *rush.Router) {
        gr2.Use(middleware2)      // Inherits middleware1, adds middleware2
        
        gr2.Get("/level2", handler2)  // Gets: middleware1 + middleware2
        
        gr2.Use(middleware3)
        
        gr2.Get("/level3", handler3)  // Gets: middleware1 + middleware2 + middleware3
    })
    
    gr1.Use(middleware4)
    gr1.Get("/back-to-level1", handler4)  // Gets: middleware1 + middleware4
})
```

### Single-Route Middleware with `.With()`

For applying middleware to individual routes without affecting the group or global scope:

```go
// In root level
r.With(specialMiddleware).Get("/special", handler)  // Only this route gets specialMiddleware
r.Get("/normal", normalHandler)                     // No additional middleware

// In a group
r.Group(func(r *rush.Router) {
    r.Use(groupMiddleware)
    
    r.Get("/regular", handler1)                           // Gets: groupMiddleware
    r.With(extraMiddleware).Post("/enhanced", handler2)   // Gets: groupMiddleware + extraMiddleware
    r.Get("/regular2", handler3)                          // Gets: groupMiddleware (not affected by .With())
})
```

### Middleware Execution Examples

#### Example 1: Order Matters
```go
r.Group(func(r *rush.Router) {
    r.Use(auth)          // Line 1
    r.Get("/a", h1)      // Gets: auth
    r.Use(logging)       // Line 3  
    r.Get("/b", h2)      // Gets: auth + logging
    r.Use(validation)    // Line 5
    r.Get("/c", h3)      // Gets: auth + logging + validation
})
```

#### Example 2: Global + Group Interaction
```go
r.Use(globalCORS)       // Global - wraps everything
r.Use(globalLogging)    // Global - wraps everything

r.Group(func(r *rush.Router) {
    r.Use(groupAuth)
    r.Get("/protected", handler)  
    // Execution: globalCORS → globalLogging → groupAuth → handler
})

r.Get("/public", handler)
// Execution: globalCORS → globalLogging → handler
```

#### Example 3: Complex Nesting
```go
r.Use(global1)

r.GroupWithPrefix("/api", func(api *rush.Router) {
    api.Use(apiMiddleware)
    
    api.GroupWithPrefix("/v1", func(v1 *rush.Router) {
        v1.Use(v1Middleware)
        v1.Get("/users", handler)  // Route: /api/v1/users
        // Execution: global1 → apiMiddleware → v1Middleware → handler
    })
})
```

## Route Matching & Precedence

Rush prioritizes routes based on specificity, ensuring the most specific match is always chosen:

1. **Exact Match** – Static routes match exactly
   - `/users/new` matches only `/users/new`

2. **Parameter Match** – Routes with named parameters
   - `/users/{id}` matches `/users/123` and extracts `id = "123"`

3. **Wildcard Match** – Catch-all routes (fallback)
   - `/users/*` matches any path starting with `/users/`

### Matching Examples

Given these registered routes:
```go
r.Get("/users/new", handler1)        // Exact match
r.Get("/users/{id}", handler2)       // Parameter match  
r.Get("/users/{id}/edit", handler3)  // Parameter + exact
r.Get("/users/*", handler4)          // Wildcard fallback
```

**Request matching behavior:**

| Request Path           | Matched Route         | Extracted Params | Priority |
|------------------------|-----------------------|------------------|----------|
| `/users/new`           | `/users/new`          | None             | Exact    |
| `/users/123`           | `/users/{id}`         | `id="123"`       | Parameter |
| `/users/123/edit`      | `/users/{id}/edit`    | `id="123"`       | Parameter + Exact |
| `/users/123/settings`  | `/users/*`            | None             | Wildcard |
| `/users/new/something` | `/users/*`            | None             | Wildcard |

## Custom Error & Default Handlers

Rush provides customizable handlers for common HTTP scenarios:

- **`r.NotFound`**: Custom 404 Not Found responses
- **`r.MethodNotAllowed`**: Custom 405 Method Not Allowed responses  
- **`r.AutoOptions`**: Global OPTIONS request handler (used automatically when no specific OPTIONS route is registered for a path)

**Important**: These handlers automatically inherit all global middleware applied to the router.

```go
// Custom 404 handler
r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte(`{"error": "Resource not found"}`))
})

// Custom global OPTIONS handler
r.AutoOptions = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
    w.WriteHeader(http.StatusOK)
})
```

## Trailing Slash Behavior

Rush **normalizes all paths internally**, treating `/foo` and `/foo/` as the same route:

- **Default**: No redirects, both paths serve the same handler
- **With `RedirectTrailingSlash = true`**: Requests to `/foo/` redirect to `/foo`

This approach:
- Simplifies route management (no duplicate routes needed)
- Provides optional SEO-friendly redirects when needed

```go
r.RedirectTrailingSlash = true
r.Get("/users", handler)

// GET /users/   → 301 redirect to /users
// GET /users    → serves handler normally
```

## Common Patterns

### Static File Serving
```go
r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))), "GET")
```

### API Versioning
```go
r.GroupWithPrefix("/api/v1", func(r *rush.Router) {
    r.Use(authMiddleware)
    r.Get("/users", v1ListUsers)
    r.Post("/users", v1CreateUser)
})

r.GroupWithPrefix("/api/v2", func(r *rush.Router) {
    r.Use(authMiddleware, rateLimitMiddleware)
    r.Get("/users", v2ListUsers)
    r.Post("/users", v2CreateUser)
})
```

### Protected Routes with Middleware
```go
// Public routes
r.Get("/", publicHandler)
r.Post("/login", loginHandler)

// Protected admin routes
r.Group(func(r *rush.Router) {
    r.Use(authMiddleware, adminMiddleware)
    r.Get("/admin/dashboard", adminDashboard)
    r.Post("/admin/users", createAdminUser)
})
```

## Important Notes

### Middleware Registration Rules
- **Global middleware** affects everything and should be registered before any routes
- **Group middleware** only affects routes defined **after** the middleware registration within that group
- **Route order matters** within groups - middleware accumulates as you go down
- Use `.With()` for one-off middleware needs without affecting the group

### Route Pattern Rules
- **Wildcards** (`*`) can only be used at the end of patterns
- **Parameter names** must be non-empty: `{id}` ✅, `{}` ❌
- **Paths are normalized** internally (trailing slashes removed)
- **Parameters are strings** - convert types as needed in handlers
