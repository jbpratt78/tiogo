package middleware

import (
	"context"
	"fmt"
	"github.com/go-chi/chi"
	"net/http"
	"strings"
)

// Contexts extract all of the params related to their route
type contextMapKey string

func (c contextMapKey) String() string {
	return "pkg.server.context" + string(c)
}

// ContextMapKey is the key to the request context
var ContextMapKey = contextMapKey("ctxMap")

// ContextMap extract from request and type asserts it (helper function.)
func ContextMap(r *http.Request) map[string]string {
	return (r.Context().Value(ContextMapKey)).(map[string]string)
}

var DefaultCacheSkipOnHit = true
var DefaultWriteOnReturn = true

// InitialCtx runs for every route, sets the response to JSON for all responses and unpacks AccessKey&SecretKey
func InitialCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		ctxMap := make(map[string]string)

		xKeys := strings.Split(r.Header.Get("X-ApiKeys"), ";")
		for x := range xKeys {
			keys := strings.Split(xKeys[x], "=")
			switch {
			case strings.ToLower(keys[0]) == "accesskey":
				ctxMap["AccessKey"] = keys[1]

			case strings.ToLower(keys[0]) == "secretkey":
				ctxMap["SecretKey"] = keys[1]
			}
		}

		ctxMap["SkipOnHit"] = r.Header.Get("X-Cache-SkipOnHit")
		ctxMap["WriteOnReturn"] = r.Header.Get("X-Cache-WriteOnReturn")

		if ctxMap["SkipOnHit"] == "" {
			ctxMap["SkipOnHit"] = fmt.Sprintf("%v", DefaultCacheSkipOnHit)
		}
		if ctxMap["WriteOnReturn"] == "" {
			ctxMap["WriteOnReturn"] = fmt.Sprintf("%v", DefaultWriteOnReturn)
		}

		ctx := context.WithValue(r.Context(), ContextMapKey, ctxMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// AccessKey is the Tenable.io access key required in the header
func AccessKey(r *http.Request) string {
	return ContextMap(r)["AccessKey"]
}

func SkipOnHit(r *http.Request) string {
	return ContextMap(r)["SkipOnHit"]
}

func WriteOnReturn(r *http.Request) string {
	return ContextMap(r)["WriteOnReturn"]
}

// SecretKey is the Tenable.io secret key required in the header
func SecretKey(r *http.Request) string {
	return ContextMap(r)["SecretKey"]
}

// ExportUUID is used for Vulns and Asset exports
func ExportUUID(r *http.Request) string {
	return ContextMap(r)["ExportUUID"]
}

func ScannerID(r *http.Request) string {
	return ContextMap(r)["ScannerID"]
}
func GroupID(r *http.Request) string {
	return ContextMap(r)["GroupID"]
}
func AgentID(r *http.Request) string {
	return ContextMap(r)["AgentID"]
}

func Offset(r *http.Request) string {
	return ContextMap(r)["Offset"]
}
func Limit(r *http.Request) string {
	return ContextMap(r)["Limit"]
}

// CHunkID ...
func ChunkID(r *http.Request) string {
	return ContextMap(r)["ChunkID"]
}

func ExportCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxMap := r.Context().Value(ContextMapKey).(map[string]string)
		ctxMap["ExportUUID"] = chi.URLParam(r, "ExportUUID")
		ctx := context.WithValue(r.Context(), ContextMapKey, ctxMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ExportChunkCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxMap := r.Context().Value(ContextMapKey).(map[string]string)
		ctxMap["ChunkID"] = chi.URLParam(r, "ChunkID")
		ctx := context.WithValue(r.Context(), ContextMapKey, ctxMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ScannersCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxMap := r.Context().Value(ContextMapKey).(map[string]string)
		ctxMap["ScannerID"] = chi.URLParam(r, "ScannerID")
		ctxMap["Offset"] = r.URL.Query().Get("offset")
		ctxMap["Limit"] = r.URL.Query().Get("limit")

		if ctxMap["Offset"] == "" {
			ctxMap["Offset"] = "0"
		}
		if ctxMap["Limit"] == "" {
			ctxMap["Limit"] = "5000"
		}
		ctx := context.WithValue(r.Context(), ContextMapKey, ctxMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func AgentGroupCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxMap := r.Context().Value(ContextMapKey).(map[string]string)
		ctxMap["GroupID"] = chi.URLParam(r, "GroupID")
		ctx := context.WithValue(r.Context(), ContextMapKey, ctxMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func AgentCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxMap := r.Context().Value(ContextMapKey).(map[string]string)
		ctxMap["AgentID"] = chi.URLParam(r, "AgentID")
		ctx := context.WithValue(r.Context(), ContextMapKey, ctxMap)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
