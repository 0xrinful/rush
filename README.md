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
- Lightweight, zero dependencies, and easy-to-read codebase (~280 LOC).

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

    // Register global middleware (applied to all routes, including error handlers)
    // Middleware execution order: first registered runs first
    r.Use(loggingMiddleware)    // runs first
    r.Use(authMiddleware)       // runs second
    r.Use(corsMiddleware)       // runs third

    // Basic routes with single HTTP method
    r.Get("/", homeHandler)
    r.Get("/users", listUsersHandler)
    r.Post("/users", createUserHandler)

    // Routes with multiple HTTP methods
    r.HandleFunc("/users/{id}", userHandler, "GET", "PUT", "DELETE")

    // Named parameters - access via r.PathValue("id") in handler
    r.Get("/users/{id}/profile", func(w http.ResponseWriter, r *http.Request) {
        userID := r.PathValue("id") // Extract the parameter
        w.Write([]byte("User ID: " + userID))
    })

    // Wildcard routes (only at the end of pattern)
    r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))), "GET")

   // Route grouping with shared middleware
    r.Group(func(r *rush.Router) {
        r.Use(adminAuthMiddleware) // Only applies to routes in this group
        r.Get("/admin/dashboard", adminDashboardHandler)
        r.Post("/admin/users", adminCreateUserHandler)
        
        // Nested groups supported
        r.Group(func(r *rush.Router) {
            r.Use(superAdminMiddleware) // Additional middleware for nested group
            r.Delete("/admin/system/reset", systemResetHandler)
        })
    }) 

    // Prefix grouping for API versioning
    r.GroupWithPrefix("/api/v1", func(r *rush.Router) {
         r.Use(apiMiddleware)
         r.Get("/users/{id}", apiGetUserHandler)     // Full route: /api/v1/users/{id}
         r.Post("/users", apiCreateUserHandler)      // Full route: /api/v1/users
     })

    // Start HTTP server
    http.ListenAndServe(":8080", r)
}

// Example custom error handler
func custom404Handler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotFound)
    w.Write([]byte("Custom 404: Page not found"))
}

// Example handlers (implement these in your application)
func homeHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Welcome to Rush Router!"))
}

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
    // Implementation...
}
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

## Middleware System

### Global Middleware

Middleware registered with `r.Use()` applies to **all routes**, including custom error handlers:

```go
r.Use(middleware1)  // Runs first
r.Use(middleware2)  // Runs second
r.Use(middleware3)  // Runs third (closest to handler)

// Execution order: middleware1 → middleware2 → middleware3 → handler → middleware3 → middleware2 → middleware1
```

### Group Middleware

Middleware can be scoped to specific route groups:

```go
r.Use(globalMiddleware) // Applies to all routes

r.Group(func(r *rush.Router) {
    r.Use(groupMiddleware) // Only applies to routes in this group
    r.Get("/protected", handler) // Uses: globalMiddleware → groupMiddleware → handler
})

r.Get("/public", handler) // Uses: globalMiddleware → handler
```

### Prefix Groups

Combine path prefixes with group-specific middleware:

```go
r.GroupWithPrefix("/api/v1", func(r *rush.Router) {
    r.Use(apiVersionMiddleware)
    r.Use(rateLimitMiddleware)
    
    r.Get("/users", handler)     // Route: /api/v1/users
    r.Post("/posts", handler)    // Route: /api/v1/posts
})
```

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

## Important Notes & Limitations

### Middleware Registration Order
- Middleware must be registered **before** routes that should use it
- Routes defined after middleware registration inherit that middleware
- Group middleware is additive to global middleware

```go
r.Use(middleware1)
r.Get("/foo", handler)    // Uses: middleware1
r.Use(middleware2)
r.Get("/bar", handler)    // Uses: middleware1 + middleware2
```

### Route Pattern Rules
- **Wildcards** (`*`) can only be used at the end of patterns
- **Parameter names** must be non-empty: `{id}` ✅, `{}` ❌
- **Paths are normalized** internally (trailing slashes removed)
- **Parameters are strings** - convert types as needed in handlers
