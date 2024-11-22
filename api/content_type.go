// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"net/http"
	"strings"
)

const (
	contentTypeHeader = "Content-Type"
	plainContentType  = "text/plain; charset=utf-8"
	jsonContentType   = "application/json" // Default content type
)

// ContentTypeRule defines a rule for content type determination
type contentTypeRule struct {
	path        string
	method      string
	contentType string
}

var contentTypeRules = []contentTypeRule{
	{
		path:        "/v1/snapshot",
		method:      http.MethodGet,
		contentType: "application/x-gzip",
	},
	{
		path:        "/v1/snapshot",
		method:      http.MethodPut,
		contentType: "application/octet-stream",
	},
	{
		path:        "/v1/kv",
		method:      http.MethodPut,
		contentType: "application/octet-stream",
	},
	{
		path:        "/v1/kv",
		method:      http.MethodDelete,
		contentType: "",
	},
	{
		path:        "/v1/kv",
		method:      http.MethodGet,
		contentType: "",
	},
	{
		path:        "/v1/event/fire",
		method:      http.MethodPut,
		contentType: "application/octet-stream",
	},
	{
		path:        "/ui",
		method:      http.MethodGet,
		contentType: plainContentType,
	},
}

// SetContentType sets the request and response Content-Type header
func SetContentType(req *http.Request) string {
	// If the request is nil, returns the default content type
	if req == nil {
		return plainContentType
	}

	if isIndexPage(req) {
		return plainContentType
	}

	if strings.HasPrefix(req.URL.Path, "/v1/internal") {
		return req.Header.Get(contentTypeHeader)
	}

	// Check against defined endpoint and required content type rules
	for _, rule := range contentTypeRules {
		if matchesRule(req, rule) {
			return rule.contentType
		}
	}

	// Default to JSON for all other endpoints
	return jsonContentType
}

// matchesRule checks if a request matches a content type rule
func matchesRule(req *http.Request, rule contentTypeRule) bool {
	return strings.HasPrefix(req.URL.Path, rule.path) &&
		(rule.method == "" || req.Method == rule.method)
}

// isIndexPage checks if the request is for the index page
func isIndexPage(req *http.Request) bool {
	return req.URL.Path == "/"
}
