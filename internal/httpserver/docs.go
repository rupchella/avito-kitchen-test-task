package httpserver

import "net/http"

const swaggerUIPage = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>Avito Kitchen API</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
    window.onload = function () {
        window.ui = SwaggerUIBundle({
            url: "/openapi.yaml",
            dom_id: "#swagger-ui",
            deepLinking: true
        });
    };
</script>
</body>
</html>
`

func SwaggerUI(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusOK)

	_, _ = writer.Write([]byte(swaggerUIPage))
}

func OpenAPISpec(spec []byte) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Type", "application/yaml; charset=utf-8")
		writer.WriteHeader(http.StatusOK)

		_, _ = writer.Write(spec)
	}
}
