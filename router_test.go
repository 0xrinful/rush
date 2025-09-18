package rush

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_Matching(t *testing.T) {
	tests := []struct {
		methods []string
		pattern string

		reqMethod string
		reqPath   string

		expectedStatus int
		expectedParams map[string]string
	}{
		// exact match
		{
			[]string{"GET"},
			"/ok",

			"GET", "/ok",
			http.StatusOK, nil,
		},

		{
			[]string{"GET"},
			"/ok",

			"GET", "/not",
			http.StatusNotFound, nil,
		},

		{
			[]string{"GET"},
			"/api/v1/status/ping",

			"GET", "/api/v1/status/ping",
			http.StatusOK, nil,
		},

		{
			[]string{"GET"},
			"/api/v1/status/ping",

			"GET", "/api/v1/status",
			http.StatusNotFound, nil,
		},

		{
			[]string{"GET"},
			"/api/v1/status",

			"GET", "/api/v1/status/ping",
			http.StatusNotFound, nil,
		},

		// empty segments
		{
			[]string{"GET"},
			"/api/v1/status",

			"GET", "/api///v1///status",
			http.StatusOK, nil,
		},

		// path params
		{
			[]string{"GET"},
			"/user/{id}",

			"GET", "/user/12",
			http.StatusOK,
			map[string]string{"id": "12"},
		},

		{
			[]string{"GET"},
			"/user/{id}",

			"GET", "/user/12/profile",
			http.StatusNotFound, nil,
		},

		{
			[]string{"GET"},
			"/user/{name}/{age}",

			"GET", "/user/name1/age1",
			http.StatusOK,
			map[string]string{"name": "name1", "age": "age1"},
		},

		{
			[]string{"GET"},
			"/user/{name}/{age}",

			"GET", "/user/12",
			http.StatusNotFound, nil,
		},

		// all methods
		{
			[]string{},
			"/ok",
			"GET", "/ok",
			http.StatusOK, nil,
		},
		{
			[]string{},
			"/ok",
			"DELETE", "/ok",
			http.StatusOK, nil,
		},

		// head requests
		{
			[]string{"GET"},
			"/ok",

			"HEAD", "/ok",
			http.StatusOK, nil,
		},
		{
			[]string{"HEAD"},
			"/ok",

			"HEAD", "/ok",
			http.StatusOK, nil,
		},
		{
			[]string{"HEAD"},
			"/ok",

			"GET", "/ok",
			http.StatusMethodNotAllowed, nil,
		},
	}

	for _, tt := range tests {
		r := New()

		handler := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		r.HandleFunc(tt.pattern, handler, tt.methods...)

		rq := httptest.NewRequest(tt.reqMethod, tt.reqPath, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)

		if w.Code != tt.expectedStatus {
			t.Fatalf(
				"[%v %v] pattern=%q methods=%v → expected status %d, got %d",
				tt.reqMethod, tt.reqPath, tt.pattern, tt.methods,
				tt.expectedStatus, w.Code,
			)
		}

		if len(tt.expectedParams) > 0 {
			for k, want := range tt.expectedParams {
				got := rq.PathValue(k)
				if got != want {
					t.Errorf(
						"[%v %v] pattern=%q methods=%v → PathValue(%q): expected %q, got %q",
						tt.reqMethod, tt.reqPath, tt.pattern, tt.methods,
						k, want, got,
					)
				}
			}
		}
	}
}

func TestRouter_Overlap(t *testing.T) {
	r := New()

	r.Get("/users/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("wildcard"))
	})
	r.Get("/users/delete/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("param"))
	})
	r.Get("/users/new", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("static"))
	})

	tests := []struct {
		reqPath string
		want    string
	}{
		{"/users/new", "static"},
		{"/users/new/", "static"},
		{"/users/delete/23", "param"},
		{"/users/delete%2F23", "param"},
		{"/users/delete", "wildcard"},
		{"/users/delete/23/foo", "wildcard"},
		{"/users/other", "wildcard"},
		{"/users/new/other", "wildcard"},
	}

	for _, tt := range tests {
		rq := httptest.NewRequest(http.MethodGet, tt.reqPath, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)
		body := w.Body.String()

		if body != tt.want {
			t.Errorf("path %q: expected body %q, got %q", tt.reqPath, tt.want, body)
		}
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	r := New()

	allowedMethods := []string{http.MethodGet, http.MethodDelete, http.MethodPut}
	allow := "DELETE, GET, HEAD, OPTIONS, PUT"
	handler := func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }
	r.HandleFunc("/ok", handler, allowedMethods...)

	tests := []struct {
		method string
		want   int
	}{
		{"GET", http.StatusOK},
		{"DELETE", http.StatusOK},
		{"PUT", http.StatusOK},
		{"POST", http.StatusMethodNotAllowed},
		{"PATCH", http.StatusMethodNotAllowed},
		{"HEAD", http.StatusOK},           // auto-handled
		{"OPTIONS", http.StatusNoContent}, // auto-handled
	}

	for _, tt := range tests {
		rq := httptest.NewRequest(tt.method, "/ok", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)
		if w.Code != tt.want {
			t.Errorf("method %q: expected status %d, got %d", tt.method, tt.want, w.Code)
		}

		if w.Code == http.StatusMethodNotAllowed {
			got := w.Header().Get("Allow")
			if got != allow {
				t.Errorf("method %q: expected Allow header %q, got %q", tt.method, allow, got)
			}
		}
	}
}

func TestRouter_Options(t *testing.T) {
	r := New()
	allowedMethods := []string{http.MethodGet, http.MethodPut}

	r.HandleFunc("/ok", nil, allowedMethods...)

	tests := []struct {
		path  string
		code  int
		allow string
	}{
		{"/ok", http.StatusNoContent, "GET, HEAD, OPTIONS, PUT"},
		{"/not", http.StatusNotFound, ""},
	}

	for _, tt := range tests {
		rq := httptest.NewRequest(http.MethodOptions, tt.path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)
		if w.Code != tt.code {
			t.Fatalf("path %q: expected status %d, got %d", tt.path, tt.code, w.Code)
		}

		if tt.allow != "" {
			got := w.Header().Get("Allow")
			if got != tt.allow {
				t.Errorf("expected Allow header %q, got %q", tt.allow, got)
			}
		}
	}
}

func TestRouter_Middleware(t *testing.T) {
	used := ""

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			used += "1"
			next.ServeHTTP(w, r)
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			used += "2"
			next.ServeHTTP(w, r)
		})
	}

	m3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			used += "3"
			next.ServeHTTP(w, r)
		})
	}

	m4 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			used += "4"
			next.ServeHTTP(w, r)
		})
	}

	m5 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			used += "5"
			next.ServeHTTP(w, r)
		})
	}

	m6 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			used += "6"
			next.ServeHTTP(w, r)
		})
	}

	handler := func(w http.ResponseWriter, r *http.Request) {}

	r := New()
	r.Use(m1, m2)
	r.Use(m3)

	r.Get("/1", handler)

	r.Group(func(r *Router) {
		r.Use(m4)
		r.Get("/2", handler)

		r.Group(func(r *Router) {
			r.Use(m5)
			r.Get("/3", handler)
		})
	})

	r.Group(func(r *Router) {
		r.Use(m6)
		r.Get("/4", handler)
	})

	r.Get("/5", handler)

	tests := []struct {
		reqMethod string
		reqPath   string
		used      string
		status    int
	}{
		{
			"GET", "/1",
			"123", http.StatusOK,
		},

		{
			"GET", "/2",
			"1234", http.StatusOK,
		},

		{
			"GET", "/3",
			"12345", http.StatusOK,
		},

		{
			"GET", "/4",
			"1236", http.StatusOK,
		},

		{
			"GET", "/5",
			"123", http.StatusOK,
		},

		// Check top-level middleware used on the 404/405/OPTIONS handlers
		{
			"GET", "/notfound",
			"123", http.StatusNotFound,
		},

		{
			"POST", "/1",
			"123", http.StatusMethodNotAllowed,
		},

		{
			"OPTIONS", "/1",
			"123", http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		used = ""
		rq := httptest.NewRequest(tt.reqMethod, tt.reqPath, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, rq)

		if w.Code != tt.status {
			t.Errorf(
				"%s %s: expected status %d, got %d", tt.reqMethod, tt.reqPath, tt.status, w.Code,
			)
		}

		if used != tt.used {
			t.Errorf(
				"%s %s: expected middleware order %q, got %q",
				tt.reqMethod, tt.reqPath, tt.used, used,
			)
		}
	}
}
