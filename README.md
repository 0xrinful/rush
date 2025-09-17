# Rush Router

Rush is a fast, lightweight HTTP router for Go with named parameters, wildcards, customizable handlers, and flexible middleware, all in ~280 LOC.

## Features

- Use named parameters (e.g., `/user/{id}` via `r.PathValue("id")`) and wildcards (e.g., `/users/*`) in routes.
- Supports overlapping routes, choosing the most specific match when multiple patterns could apply.
- Middleware grouping with inheritance and prefix support (similar to chi).
- Customizable handlers for `404 Not Found`, `405 Method Not Allowed`, and `OPTIONS`.
- Automatic handling of `OPTIONS` and `HEAD` requests.
- Compatible with `http.Handler`, `http.HandlerFunc`, and standard Go middleware.
- Lightweight, zero dependencies, and easy-to-read codebase.

## Installation

Install Rush with:

```
go get github.com/0xrinful/rush
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

    // Custom handlers for common HTTP errors
    r.NotFound = custom404Handler
    r.MethodNotAllowed = custom405Handler
    r.Options = customOptionsHandler

    // Register global middleware (applied to all routes, including error handlers and OPTIONS responses)
    r.Use(globalMiddleware1)

    // Define routes with a single HTTP method
    r.Get("/users/{id}", getUserHandler)

    // Define routes with multiple HTTP methods
    r.HandleFunc("/user/{id}", updateUserHandler, "GET", "POST")

    // Parameterized route: in the handler, access the "id" parameter using r.PathValue("id")
    // Note: this `r` is the *http.Request inside the handler, not the router `r` used to define routes
    r.Delete("/users/delete/{id}", deleteUserHandler)

    // Wildcard route (can only be used at the end of the pattern)
    r.Get("/users/*", usersWildcardHandler)

    // Route grouping
    r.Group(func(r *rush.Router) {
        // Middleware declared inside the group applies only to routes within the group
        r.Use(groupMiddleware1)
        r.Post("/user", createUserHandler)

        // Nested groups are supported
        r.Group(func(r *rush.Router) {
            r.Use(groupMiddleware2)
        })
    })

    // Prefix grouping
    r.GroupWithPrefix("/v1", func(r *rush.Router) {
        r.Use(apiMiddleware)
        r.Get("/user/{id}", apiGetUserHandler) // full route: /v1/user/{id}
    })

    // Start HTTP server
    http.ListenAndServe(":8080", r)
}
```

### Pattern Overlap & Matching

Rush prioritizes routes based on specificity:

1. **Exact match** – Matches the route exactly.  
   Example: `/users/new` always matches `/users/new`, not `/users/*`.
2. **Parameter match** – Matches routes with named parameters and extracts values.  
   Example: `/users/delete/{id}` matches `/users/delete/5` and extracts `id = 5`.
3. **Wildcard match** – Matches any remaining paths at the end of a pattern.  
   Example: `/users/*` matches `/users/foo` if no exact or parameter route matches.


**Example routes:**
- /users/*
- /users/new
- /users/delete/{id}

**Matching behavior:**

| Request            | Matched Route          | Notes                         |
|--------------------|------------------------|-------------------------------|
| `/users/new`       | `/users/new`           | Exact match wins              |
| `/users/delete/5`  | `/users/delete/{id}`   | Parameter match               |
| `/users/profile`   | `/users/*`             | Wildcard used as fallback     |

### Notes

* Middleware must be registered **before** the routes that should use it. Any middleware added **after** a route will not affect that route.  

Example:

```go
r := rush.New()
r.Use(middleware1)
r.HandleFunc("/foo", handler) // This route uses middleware1 only
r.Use(middleware2)
r.HandleFunc("/bar", handler) // This route uses both middleware1 and middleware2
```
