// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"net/http"
	"strings"
)

const (
	ContentTypeHeader = "Content-Type"
	PlainContentType  = "text/plain; charset=utf-8"
	JSONContentType   = "application/json" // Default content type
)

// ContentTypeRule defines a rule for content type determination
type ContentTypeRule struct {
	Path        string
	Method      string
	ContentType string
}

var contentTypeRules = []ContentTypeRule{
	{
		Path:        "/v1/snapshot",
		Method:      http.MethodGet,
		ContentType: "application/x-gzip",
	},
	{
		Path:        "/v1/snapshot",
		Method:      http.MethodPut,
		ContentType: "application/octet-stream",
	},
	{
		Path:        "/v1/kv",
		Method:      http.MethodPut,
		ContentType: "application/octet-stream",
	},
	{
		Path:        "/v1/kv",
		Method:      http.MethodDelete,
		ContentType: "",
	},
	{
		Path:        "/v1/kv",
		Method:      http.MethodGet,
		ContentType: "",
	},
	{
		Path:        "/v1/event/fire",
		Method:      http.MethodPut,
		ContentType: "application/octet-stream",
	},
}

// DetermineContentType returns the appropriate content type based on the request
// If the request is nil, returns the default content type
func DetermineContentType(req *http.Request) string {
	if req == nil {
		return PlainContentType
	}

	if strings.HasPrefix(req.URL.Path, "/v1/internal") {
		return req.Header.Get(ContentTypeHeader)
	}

	// Check against defined endpoint and required content type rules
	for _, rule := range contentTypeRules {
		if matchesRule(req, rule) {
			return rule.ContentType
		}
	}

	// Default to JSON for all other endpoints
	return JSONContentType
}

// matchesRule checks if a request matches a content type rule
func matchesRule(req *http.Request, rule ContentTypeRule) bool {
	return strings.HasPrefix(req.URL.Path, rule.Path) &&
		(rule.Method == "" || req.Method == rule.Method)
}
