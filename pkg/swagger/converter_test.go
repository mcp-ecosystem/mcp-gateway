package swagger

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConverter_Convert(t *testing.T) {
	converter := NewConverter()

	// Test with a simple Swagger 2.0 specification
	spec := `{
		"swagger": "2.0",
		"info": {
			"title": "Test API",
			"description": "Test API description",
			"version": "1.0.0"
		},
		"host": "api.example.com",
		"basePath": "/v1",
		"schemes": ["https"],
		"paths": {
			"/test": {
				"get": {
					"summary": "Test endpoint",
					"responses": {
						"200": {
							"description": "Successful response"
						}
					}
				},
				"options": {
					"summary": "CORS options",
					"responses": {
						"200": {
							"description": "CORS headers"
						}
					}
				}
			},
			"/users/{userId}": {
				"get": {
					"summary": "Get user by ID",
					"parameters": [
						{
							"name": "userId",
							"in": "path",
							"required": true,
							"type": "string",
							"description": "User ID"
						}
					],
					"responses": {
						"200": {
							"description": "User found"
						}
					}
				}
			}
		}
	}`

	config, err := converter.ConvertFromJSON([]byte(spec))
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify the converted configuration
	assert.Equal(t, "Test API", config.Name)
	assert.Equal(t, 1, len(config.Routers))
	assert.NotNil(t, config.Routers[0].CORS)

	assert.Equal(t, 1, len(config.Servers))
	assert.Equal(t, "Test API", config.Servers[0].Name)
	assert.Equal(t, "Test API description", config.Servers[0].Description)
	assert.Equal(t, "https://api.example.com/v1", config.Servers[0].Config["url"])

	// Verify tools
	assert.GreaterOrEqual(t, len(config.Tools), 2)
}

func TestConverter_ConvertFromYAML(t *testing.T) {
	converter := NewConverter()

	// Test with a simple Swagger specification in YAML
	spec := `swagger: '2.0'
info:
  title: Test API
  description: Test API description
  version: 1.0.0
host: api.example.com
basePath: /v1
schemes:
  - https
paths:
  /test:
    get:
      summary: Test endpoint
      responses:
        200:
          description: Successful response
    options:
      summary: CORS options
      responses:
        200:
          description: CORS headers`

	config, err := converter.ConvertFromYAML([]byte(spec))
	assert.NoError(t, err)
	assert.NotNil(t, config)

	// Verify the converted configuration
	assert.Equal(t, "Test API", config.Name)
	assert.Equal(t, 1, len(config.Routers))
	assert.Equal(t, 1, len(config.Servers))
	assert.Equal(t, "Test API", config.Servers[0].Name)
	assert.Equal(t, "Test API description", config.Servers[0].Description)
	assert.Equal(t, "https://api.example.com/v1", config.Servers[0].Config["url"])
}
