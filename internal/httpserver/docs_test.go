package httpserver

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerUIServesPage(t *testing.T) {
	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/docs",
		nil,
	)
	recorder := httptest.NewRecorder()

	SwaggerUI(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")

	if !strings.HasPrefix(contentType, "text/html") {
		t.Errorf("unexpected content type: %q", contentType)
	}

	if !strings.Contains(recorder.Body.String(), "/openapi.yaml") {
		t.Error("page must load the specification from /openapi.yaml")
	}
}

func TestOpenAPISpecServesDocument(t *testing.T) {
	spec := []byte("openapi: 3.1.0\n")

	request := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodGet,
		"/openapi.yaml",
		nil,
	)
	recorder := httptest.NewRecorder()

	OpenAPISpec(spec)(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	contentType := recorder.Header().Get("Content-Type")

	if !strings.HasPrefix(contentType, "application/yaml") {
		t.Errorf("unexpected content type: %q", contentType)
	}

	if recorder.Body.String() != string(spec) {
		t.Errorf("unexpected body: %q", recorder.Body.String())
	}
}

// Без файла спецификации документация не публикуется:
// сервис должен работать и без неё.
func TestRouterDisablesDocsWithoutSpec(t *testing.T) {
	tests := []struct {
		name       string
		spec       []byte
		target     string
		statusCode int
	}{
		{"docs without spec", nil, "/docs", http.StatusNotFound},
		{"spec without spec", nil, "/openapi.yaml", http.StatusNotFound},
		{"docs with spec", []byte("openapi: 3.1.0"), "/docs", http.StatusOK},
		{"spec with spec", []byte("openapi: 3.1.0"), "/openapi.yaml", http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			router := NewRouter(RouterDependencies{
				OpenAPISpec: test.spec,
			})

			request := httptest.NewRequestWithContext(
				t.Context(),
				http.MethodGet,
				test.target,
				nil,
			)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, request)

			if recorder.Code != test.statusCode {
				t.Fatalf(
					"expected status %d, got %d",
					test.statusCode,
					recorder.Code,
				)
			}
		})
	}
}
