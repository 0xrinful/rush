# Rush

Rush is a fast, lightweight HTTP router for Go with named parameters, wildcards, customizable handlers, and flexible middleware, all in ~280 LOC.

## Features

- **Named Parameters**: Extract values from URLs (e.g., `/user/{id}`)
- **Wildcards**: Catch-all routes (e.g., `/static/*` for file serving)
- **Route Precedence**: Automatic most-specific route matching
- **Middleware Groups**: Apply middleware to specific route groups 
- **Prefix Groups**: Group routes with common path prefixes (e.g., `/api/v1`)
- **Customizable handlers** for `404 Not Found`, `405 Method Not Allowed`, and `OPTIONS`.
- **Automatic handling** of `OPTIONS` and `HEAD` requests.
- **Standard Library Compatible**: Works with any `http.Handler` or `http.HandlerFunc`
- Lightweight, zero dependencies, and easy-to-read codebase (~300 LOC).

## Installation

Install Rush with:

```
go get github.com/0xrinful/rush@latest
```

## Quick Start Example

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
    r.Options = customOptionsHandler

    // Redirect requests with a trailing slash to the normalized route
    // Useful for SEO and consistent URLs; only affects existing routes
    r.RedirectTrailingSlash = true 

    // Global middleware - wraps the entire router (affects ALL routes and error handlers)
    r.Use(loggingMiddleware)    // outermost layer (runs first)
    r.Use(corsMiddleware)       // middle layer    (runs second)
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
            // inherits all middlewares from the parent group
            r.Use(superAdminMiddleware)
            r.Delete("/admin/system/reset", systemResetHandler)
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
## Two-Level Middleware System

Rush features a sophisticated two-level middleware system that provides both global and granular control over request processing.

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

## Error Handling

Rush provides customizable handlers for common HTTP scenarios:

- **`r.NotFound`**: Custom 404 Not Found responses
- **`r.MethodNotAllowed`**: Custom 405 Method Not Allowed responses  
- **`r.Options`**: Custom OPTIONS request handling

**Important**: These handlers automatically inherit all global middleware applied to the router.

```go
r.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte(`{"error": "Resource not found"}`))
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

### RESTful Resources
```go
r.Get("/users", listUsers)           // GET /users
r.Post("/users", createUser)         // POST /users  
r.Get("/users/{id}", getUser)        // GET /users/123
r.Put("/users/{id}", updateUser)     // PUT /users/123
r.Delete("/users/{id}", deleteUser)  // DELETE /users/123
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
