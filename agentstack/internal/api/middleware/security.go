/*
 * Copyright 2025 Raphaël MANSUY
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package middleware provides HTTP middleware for the API.
package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeadersConfig configures security headers.
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy is the Content-Security-Policy header value.
	ContentSecurityPolicy string
	// StrictTransportSecurity is the Strict-Transport-Security header value.
	StrictTransportSecurity string
	// XContentTypeOptions is the X-Content-Type-Options header value.
	XContentTypeOptions string
	// XFrameOptions is the X-Frame-Options header value.
	XFrameOptions string
	// XXSSProtection is the X-XSS-Protection header value.
	XXSSProtection string
	// ReferrerPolicy is the Referrer-Policy header value.
	ReferrerPolicy string
	// PermissionsPolicy is the Permissions-Policy header value.
	PermissionsPolicy string
	// CrossOriginEmbedderPolicy is the Cross-Origin-Embedder-Policy header value.
	CrossOriginEmbedderPolicy string
	// CrossOriginOpenerPolicy is the Cross-Origin-Opener-Policy header value.
	CrossOriginOpenerPolicy string
	// CrossOriginResourcePolicy is the Cross-Origin-Resource-Policy header value.
	CrossOriginResourcePolicy string
}

// DefaultSecurityHeadersConfig returns secure defaults for production.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains; preload",
		XContentTypeOptions:       "nosniff",
		XFrameOptions:             "DENY",
		XXSSProtection:            "1; mode=block",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
		CrossOriginEmbedderPolicy: "require-corp",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// APISecurityHeadersConfig returns security headers optimized for API endpoints.
func APISecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:     "default-src 'none'",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains",
		XContentTypeOptions:       "nosniff",
		XFrameOptions:             "DENY",
		XXSSProtection:            "0", // Disabled for APIs, can cause issues
		ReferrerPolicy:            "no-referrer",
		PermissionsPolicy:         "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()",
		CrossOriginEmbedderPolicy: "",
		CrossOriginOpenerPolicy:   "",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// SecurityHeaders returns a middleware that adds security headers.
func SecurityHeaders(config SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()

			// Content Security Policy
			if config.ContentSecurityPolicy != "" {
				h.Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// HTTP Strict Transport Security
			if config.StrictTransportSecurity != "" {
				h.Set("Strict-Transport-Security", config.StrictTransportSecurity)
			}

			// X-Content-Type-Options
			if config.XContentTypeOptions != "" {
				h.Set("X-Content-Type-Options", config.XContentTypeOptions)
			}

			// X-Frame-Options
			if config.XFrameOptions != "" {
				h.Set("X-Frame-Options", config.XFrameOptions)
			}

			// X-XSS-Protection
			if config.XXSSProtection != "" {
				h.Set("X-XSS-Protection", config.XXSSProtection)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				h.Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				h.Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// Cross-Origin-Embedder-Policy
			if config.CrossOriginEmbedderPolicy != "" {
				h.Set("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			}

			// Cross-Origin-Opener-Policy
			if config.CrossOriginOpenerPolicy != "" {
				h.Set("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			}

			// Cross-Origin-Resource-Policy
			if config.CrossOriginResourcePolicy != "" {
				h.Set("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			}

			// Remove server information headers
			h.Del("Server")
			h.Del("X-Powered-By")

			next.ServeHTTP(w, r)
		})
	}
}

// CORSConfig configures CORS behavior.
type CORSConfig struct {
	// AllowedOrigins is a list of allowed origins.
	AllowedOrigins []string
	// AllowedMethods is a list of allowed HTTP methods.
	AllowedMethods []string
	// AllowedHeaders is a list of allowed request headers.
	AllowedHeaders []string
	// ExposedHeaders is a list of headers exposed to the client.
	ExposedHeaders []string
	// AllowCredentials indicates if credentials are allowed.
	AllowCredentials bool
	// MaxAge is the max age for preflight cache in seconds.
	MaxAge int
}

// DefaultCORSConfig returns default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "X-API-Key"},
		ExposedHeaders:   []string{"X-Request-ID", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// RestrictedCORSConfig returns a restrictive CORS configuration.
func RestrictedCORSConfig(allowedOrigins []string) CORSConfig {
	return CORSConfig{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           3600, // 1 hour
	}
}

// CORS returns a middleware that handles CORS.
func CORS(config CORSConfig) func(http.Handler) http.Handler {
	allowedOrigins := make(map[string]bool)
	allowAll := false
	for _, origin := range config.AllowedOrigins {
		if origin == "*" {
			allowAll = true
		}
		allowedOrigins[origin] = true
	}

	methodsStr := strings.Join(config.AllowedMethods, ", ")
	headersStr := strings.Join(config.AllowedHeaders, ", ")
	exposedStr := strings.Join(config.ExposedHeaders, ", ")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			originAllowed := allowAll
			if !originAllowed && origin != "" {
				originAllowed = allowedOrigins[origin]
			}

			if originAllowed && origin != "" {
				if allowAll && !config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Add("Vary", "Origin")
				}

				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				if exposedStr != "" {
					w.Header().Set("Access-Control-Expose-Headers", exposedStr)
				}
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				if originAllowed {
					w.Header().Set("Access-Control-Allow-Methods", methodsStr)
					w.Header().Set("Access-Control-Allow-Headers", headersStr)
					if config.MaxAge > 0 {
						w.Header().Set("Access-Control-Max-Age", strings.TrimSpace(strings.Split(string(rune(config.MaxAge)), "")[0]))
					}
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestSanitizer sanitizes incoming requests.
func RequestSanitizer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Remove potentially dangerous headers
			r.Header.Del("X-Forwarded-Host")
			r.Header.Del("X-Original-URL")
			r.Header.Del("X-Rewrite-URL")

			// Validate Content-Type for body-carrying requests
			if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
				contentType := r.Header.Get("Content-Type")
				if contentType != "" && !isValidContentType(contentType) {
					http.Error(w, "Invalid Content-Type", http.StatusUnsupportedMediaType)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isValidContentType checks if a content type is valid.
func isValidContentType(contentType string) bool {
	validTypes := []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
		"application/xml",
		"text/xml",
	}

	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	for _, valid := range validTypes {
		if ct == valid {
			return true
		}
	}
	return false
}

// SecureResponseWriter wraps http.ResponseWriter to enforce security.
type SecureResponseWriter struct {
	http.ResponseWriter
	written bool
}

// WriteHeader sets the status code and ensures security headers are set.
func (w *SecureResponseWriter) WriteHeader(statusCode int) {
	if !w.written {
		// Ensure Content-Type is set to prevent MIME sniffing
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		w.written = true
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write writes the response body.
func (w *SecureResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// SecureResponse wraps the response writer for security.
func SecureResponse() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sw := &SecureResponseWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
		})
	}
}

// CSPBuilder helps build Content-Security-Policy headers.
type CSPBuilder struct {
	directives map[string][]string
}

// NewCSPBuilder creates a new CSP builder.
func NewCSPBuilder() *CSPBuilder {
	return &CSPBuilder{
		directives: make(map[string][]string),
	}
}

// DefaultSrc sets the default-src directive.
func (b *CSPBuilder) DefaultSrc(sources ...string) *CSPBuilder {
	b.directives["default-src"] = sources
	return b
}

// ScriptSrc sets the script-src directive.
func (b *CSPBuilder) ScriptSrc(sources ...string) *CSPBuilder {
	b.directives["script-src"] = sources
	return b
}

// StyleSrc sets the style-src directive.
func (b *CSPBuilder) StyleSrc(sources ...string) *CSPBuilder {
	b.directives["style-src"] = sources
	return b
}

// ImgSrc sets the img-src directive.
func (b *CSPBuilder) ImgSrc(sources ...string) *CSPBuilder {
	b.directives["img-src"] = sources
	return b
}

// FontSrc sets the font-src directive.
func (b *CSPBuilder) FontSrc(sources ...string) *CSPBuilder {
	b.directives["font-src"] = sources
	return b
}

// ConnectSrc sets the connect-src directive.
func (b *CSPBuilder) ConnectSrc(sources ...string) *CSPBuilder {
	b.directives["connect-src"] = sources
	return b
}

// FrameAncestors sets the frame-ancestors directive.
func (b *CSPBuilder) FrameAncestors(sources ...string) *CSPBuilder {
	b.directives["frame-ancestors"] = sources
	return b
}

// BaseURI sets the base-uri directive.
func (b *CSPBuilder) BaseURI(sources ...string) *CSPBuilder {
	b.directives["base-uri"] = sources
	return b
}

// FormAction sets the form-action directive.
func (b *CSPBuilder) FormAction(sources ...string) *CSPBuilder {
	b.directives["form-action"] = sources
	return b
}

// Build returns the CSP header value.
func (b *CSPBuilder) Build() string {
	parts := make([]string, 0, len(b.directives))
	for directive, sources := range b.directives {
		parts = append(parts, directive+" "+strings.Join(sources, " "))
	}
	return strings.Join(parts, "; ")
}
