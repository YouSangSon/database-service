# Swagger/OpenAPI Documentation Setup

This guide explains how to set up and use Swagger/OpenAPI documentation for the Database Service API.

## Prerequisites

- Go 1.21 or higher
- Active internet connection (for downloading dependencies)

## Installation Steps

### 1. Install Swagger Dependencies

```bash
# Install swag CLI tool
go install github.com/swaggo/swag/cmd/swag@latest

# Install Gin Swagger packages
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files
```

### 2. Uncomment Swagger Imports

Edit `cmd/api/main.go` and uncomment the following lines:

```go
// Change this:
// _ "github.com/YouSangSon/database-service/docs"
// ginSwagger "github.com/swaggo/gin-swagger"
// swaggerFiles "github.com/swaggo/files"

// To this:
_ "github.com/YouSangSon/database-service/docs"
ginSwagger "github.com/swaggo/gin-swagger"
swaggerFiles "github.com/swaggo/files"
```

Also uncomment the Swagger route:

```go
// Change this:
// router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

// To this:
router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
```

### 3. Generate Swagger Documentation

Run the following command from the project root:

```bash
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

This will generate:
- `docs/docs.go` - Generated documentation
- `docs/swagger.json` - OpenAPI JSON specification
- `docs/swagger.yaml` - OpenAPI YAML specification

### 4. Run the Application

```bash
# Start all services with docker-compose
docker-compose up -d

# Build and run the API server
go run cmd/api/main.go
```

### 5. Access Swagger UI

Open your browser and navigate to:

```
http://localhost:8080/swagger/index.html
```

## API Documentation Annotations

The API handlers are already annotated with Swagger comments. Here are examples:

### Main API Information (in cmd/api/main.go)

```go
// @title Database Service API
// @version 1.0
// @description Enterprise-grade database service with MongoDB/Vitess support
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
```

### Handler Examples (in internal/interfaces/http/handler/)

```go
// @Summary Create a new document
// @Description Create a new document in the specified collection
// @Tags documents
// @Accept json
// @Produce json
// @Param request body dto.CreateDocumentRequest true "Document creation request"
// @Success 201 {object} dto.CreateDocumentResponse
// @Failure 400 {object} dto.ErrorResponse
// @Router /api/v1/documents [post]
func (h *DocumentHandler) Create(c *gin.Context) {
    // Handler implementation
}
```

## Available Annotations

### Main Annotations (General API Information)

| Annotation | Description | Example |
|------------|-------------|---------|
| `@title` | API title | `@title Database Service API` |
| `@version` | API version | `@version 1.0` |
| `@description` | API description | `@description Enterprise database service` |
| `@host` | API host | `@host localhost:8080` |
| `@BasePath` | API base path | `@BasePath /api/v1` |
| `@schemes` | Supported schemes | `@schemes http https` |
| `@contact.name` | Contact name | `@contact.name API Support` |
| `@contact.url` | Contact URL | `@contact.url http://example.com` |
| `@contact.email` | Contact email | `@contact.email support@example.com` |
| `@license.name` | License name | `@license.name Apache 2.0` |
| `@license.url` | License URL | `@license.url http://www.apache.org/licenses/LICENSE-2.0.html` |

### Handler Annotations (Endpoint Information)

| Annotation | Description | Example |
|------------|-------------|---------|
| `@Summary` | Short description | `@Summary Create a document` |
| `@Description` | Detailed description | `@Description Create a new document in collection` |
| `@Tags` | Group endpoints | `@Tags documents` |
| `@Accept` | Request content type | `@Accept json` |
| `@Produce` | Response content type | `@Produce json` |
| `@Param` | Parameter definition | `@Param id path string true "Document ID"` |
| `@Success` | Success response | `@Success 200 {object} dto.Response` |
| `@Failure` | Error response | `@Failure 404 {object} dto.Error` |
| `@Router` | Route path and method | `@Router /documents/{id} [get]` |
| `@Security` | Security requirement | `@Security ApiKeyAuth` |

### Parameter Types

- `path` - Path parameter (e.g., `/documents/{id}`)
- `query` - Query parameter (e.g., `?limit=10`)
- `header` - Header parameter (e.g., `Authorization`)
- `body` - Request body
- `formData` - Form data

### Parameter Format

```
@Param name type dataType required description
```

Examples:
```go
// Path parameter
@Param id path string true "Document ID"

// Query parameter
@Param limit query int false "Limit (default 10)"

// Header parameter
@Param X-API-Key header string true "API Key"

// Body parameter
@Param request body dto.CreateDocumentRequest true "Request body"
```

## Complete Example

Here's a complete example of a documented handler:

```go
// @Summary Get document by ID
// @Description Retrieve a document from a collection by its ID
// @Tags documents
// @Accept json
// @Produce json
// @Param collection path string true "Collection name"
// @Param id path string true "Document ID"
// @Param X-API-Key header string false "API Key for authentication"
// @Success 200 {object} dto.GetDocumentResponse
// @Failure 400 {object} dto.ErrorResponse "Invalid request"
// @Failure 404 {object} dto.ErrorResponse "Document not found"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Router /documents/{collection}/{id} [get]
// @Security ApiKeyAuth
func (h *DocumentHandler) GetByID(c *gin.Context) {
    // Implementation
}
```

## Swagger UI Features

### 1. Interactive API Testing

- Click on any endpoint to expand it
- Click "Try it out" button
- Fill in parameters
- Click "Execute" to send request
- View response with status code, headers, and body

### 2. Authentication

For endpoints requiring authentication:

1. Click the "Authorize" button at the top
2. Enter your API key
3. Click "Authorize"
4. Now all requests will include the authentication header

### 3. Model Schemas

- Scroll down to see all request/response models
- Click on model names to see their structure
- Models are automatically generated from Go structs

### 4. Download Specification

You can download the OpenAPI specification in multiple formats:

- JSON: `http://localhost:8080/swagger/doc.json`
- YAML: Download from Swagger UI or access `docs/swagger.yaml`

## Regenerating Documentation

After making changes to handler annotations, regenerate the documentation:

```bash
swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
```

Or use the provided make command (if Makefile exists):

```bash
make swagger
```

## Tips and Best Practices

### 1. Keep Annotations Updated

Always update Swagger annotations when:
- Adding new endpoints
- Modifying request/response structures
- Changing parameters
- Updating error responses

### 2. Use Meaningful Descriptions

```go
// Good
@Summary Create a new document
@Description Creates a new document in the specified collection with validation

// Bad
@Summary Create
@Description Creates a document
```

### 3. Document All Response Codes

Include all possible HTTP status codes:

```go
@Success 200 {object} dto.Response "Success"
@Success 201 {object} dto.Response "Created"
@Failure 400 {object} dto.Error "Bad Request"
@Failure 401 {object} dto.Error "Unauthorized"
@Failure 403 {object} dto.Error "Forbidden"
@Failure 404 {object} dto.Error "Not Found"
@Failure 500 {object} dto.Error "Internal Server Error"
```

### 4. Group Related Endpoints

Use consistent tags to group endpoints:

```go
@Tags documents  // For document operations
@Tags health     // For health checks
@Tags admin      // For admin operations
```

### 5. Define Security Requirements

For protected endpoints:

```go
@Security ApiKeyAuth
```

Or for multiple security schemes:

```go
@Security ApiKeyAuth
@Security OAuth2
```

## Troubleshooting

### Error: "swag: command not found"

Ensure `swag` is installed and in your PATH:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

### Error: "cannot find package"

Run `go mod tidy` to download dependencies:

```bash
go mod tidy
```

### Documentation Not Updating

1. Regenerate documentation:
   ```bash
   swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal
   ```

2. Restart the application
3. Clear browser cache or try incognito mode

### 404 on /swagger/index.html

Ensure:
1. Swagger imports are uncommented in `cmd/api/main.go`
2. Swagger route is uncommented
3. Documentation has been generated with `swag init`
4. Application has been rebuilt

## Integration with CI/CD

### GitHub Actions Example

```yaml
name: Generate Swagger Docs

on:
  push:
    branches: [ main ]

jobs:
  swagger:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Set up Go
        uses: actions/setup-go@v2
        with:
          go-version: 1.21

      - name: Install swag
        run: go install github.com/swaggo/swag/cmd/swag@latest

      - name: Generate Swagger docs
        run: swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal

      - name: Commit docs
        run: |
          git config --local user.email "action@github.com"
          git config --local user.name "GitHub Action"
          git add docs/
          git diff --staged --quiet || git commit -m "Auto-generate Swagger docs"
          git push
```

## References

- [Swag GitHub Repository](https://github.com/swaggo/swag)
- [Gin Swagger](https://github.com/swaggo/gin-swagger)
- [OpenAPI Specification](https://swagger.io/specification/)
- [Swagger UI Documentation](https://swagger.io/tools/swagger-ui/)

## Example API Calls

### Using curl with Swagger

After testing in Swagger UI, you can export to curl:

```bash
# Create document
curl -X POST "http://localhost:8080/api/v1/documents" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-api-key" \
  -d '{
    "collection": "users",
    "data": {
      "name": "John Doe",
      "email": "john@example.com"
    }
  }'

# Get document
curl -X GET "http://localhost:8080/api/v1/documents/users/123" \
  -H "X-API-Key: your-api-key"

# List documents
curl -X GET "http://localhost:8080/api/v1/documents/users?limit=10&offset=0" \
  -H "X-API-Key: your-api-key"
```

## Next Steps

1. Install dependencies as shown above
2. Generate Swagger documentation
3. Start the application
4. Access Swagger UI at `http://localhost:8080/swagger/index.html`
5. Test your API endpoints interactively
6. Share the OpenAPI specification with your team
