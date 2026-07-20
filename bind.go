package gobind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"sync"
)

type Http struct {
	R *http.Request
	W http.ResponseWriter
}

// Request represents an incoming HTTP request with typed binding.
// The type parameter T specifies the request structure whose fields are populated
// from the HTTP request using struct tags (header, query, path, body, cookie).
//
// Use the Value field to access the parsed request data.
// Use the Http field to access the raw *http.Request and http.ResponseWriter.
type Request[T any] struct {
	Value T
	Http  Http
}

// Response represents an HTTP response to be serialized and written.
// The type parameter T specifies the response structure whose fields are serialized
// to the HTTP response using struct tags (header, body, cookie).
//
// Set StatusCode to control the HTTP status (default is 200).
// Set Value to the response data to be serialized.
type Response[T any] struct {
	StatusCode int
	Value      T
}

type HandlerFunc[Req any, Resp any] func(context.Context, *Request[Req]) (*Response[Resp], error)

type ErrorInterface interface {
	error
	GetStatusCode() int
	GetValue() any
}

// Error wraps an HTTP error response with a typed body that is serialized
// using struct field tags. It implements the error interface so it can be
// returned from a handler function to produce a structured error response
// with a custom status code, headers, body, and cookies.
//
// The type parameter T specifies the structure of the error body. Fields of T
// support the same binding tags as Response (header, body, cookie).
// Use NewError or construct directly by setting StatusCode and Value.
type Error[T any] struct {
	StatusCode int
	Value      T
}

func (e *Error[T]) Error() string {
	return fmt.Sprintf("%v", e.Value)
}

func (e *Error[T]) GetStatusCode() int {
	return e.StatusCode
}

func (e *Error[T]) GetValue() any {
	return e.Value
}

func NewError[T any](statusCode int, err T) *Error[T] {
	return &Error[T]{StatusCode: statusCode, Value: err}
}

type FieldBinding struct {
	// Field is the struct field metadata
	Field reflect.StructField

	// TagType is the binding source type (header, query, path, body, cookie)
	TagType string

	// TagValue is the tag value (e.g., header name, query param name)
	TagValue string
}

type StructBinding struct {
	Fields []FieldBinding
}

type BindingCache struct {
	m     *sync.Mutex
	cache map[reflect.Type]StructBinding
}

func (c *BindingCache) Set(t reflect.Type, b StructBinding) {
	c.m.Lock()
	defer c.m.Unlock()
	c.cache[t] = b
}

func (c *BindingCache) Get(t reflect.Type) (StructBinding, bool) {
	b, ok := c.cache[t]
	return b, ok
}

var (
	bindingCache = BindingCache{
		m:     &sync.Mutex{},
		cache: make(map[reflect.Type]StructBinding),
	}
)

func Handler[Req any, Resp any](next HandlerFunc[Req, Resp]) http.Handler {
	// Request
	var isReqKindPtr bool
	reqType := reflect.TypeFor[Req]()
	if reqType.Kind() == reflect.Pointer {
		isReqKindPtr = true
		reqType = reqType.Elem()
	}

	requestBinding := buildStructBinding(reqType)
	bindingCache.Set(reqType, requestBinding)

	// Response
	respType := reflect.TypeFor[Resp]()
	responseBinding := buildStructBinding(respType)
	bindingCache.Set(respType, responseBinding)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPtr := reflect.New(reqType)
		reqVal := reqPtr.Elem()

		b, ok := bindingCache.Get(reqType)
		if !ok {
			binding := buildStructBinding(reqType)
			bindingCache.Set(reqType, binding)
			b = binding
		}

		for _, field := range b.Fields {
			fieldVal := reqVal.FieldByIndex(field.Field.Index)

			if !fieldVal.CanSet() {
				continue
			}

			switch field.TagType {
			case "header":
				setFieldValue(fieldVal, field.Field, r.Header.Get(field.TagValue))
			case "query":
				setFieldValue(fieldVal, field.Field, r.URL.Query().Get(field.TagValue))
			case "path":
				setFieldValue(fieldVal, field.Field, r.PathValue(field.TagValue))
			case "body":
				if r.Body != nil {
					r.Body = setBodyValue(fieldVal, field.Field, r.Body, field.TagValue)
				}
			case "cookie":
				setFieldValue(fieldVal, field.Field, getCookieValue(r, field.TagValue))
			}
		}

		var req Req
		if isReqKindPtr {
			req = reqVal.Addr().Interface().(Req)
		} else {
			req = reqVal.Interface().(Req)
		}

		request := &Request[Req]{
			Value: req,
			Http: Http{
				R: r,
				W: w,
			},
		}

		resp, err := next(r.Context(), request)
		if err != nil {
			var gobindErr ErrorInterface
			if errors.As(err, &gobindErr) {
				writeResponse(w, r, gobindErr.GetStatusCode(), reflect.ValueOf(gobindErr.GetValue()), reflect.TypeOf(gobindErr.GetValue()))
				return
			}

			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		if resp == nil {
			http.Error(w, "nil response", http.StatusInternalServerError)
			return
		}

		gotRespType := reflect.TypeOf(resp.Value)
		if gotRespType != respType {
			panic(fmt.Sprintf("expected response type of %s, but got %s", respType.Name(), gotRespType.Name()))
		}

		respVal := reflect.ValueOf(resp.Value)

		writeResponse(w, r, resp.StatusCode, respVal, respType)
	})
}

func setFieldValue(val reflect.Value, _ reflect.StructField, value string) {
	if value == "" {
		return
	}

	if val.Kind() == reflect.Pointer {
		ptr := reflect.New(val.Type().Elem())
		setScalar(ptr.Elem(), value)
		val.Set(ptr)
		return
	}

	setScalar(val, value)
}

func setScalar(val reflect.Value, value string) {
	switch val.Kind() {
	case reflect.String:
		val.SetString(value)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err == nil {
			val.SetBool(boolVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, val.Type().Bits())
		if err == nil {
			val.SetInt(intVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, val.Type().Bits())
		if err == nil {
			val.SetUint(uintVal)
		}
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, val.Type().Bits())
		if err == nil {
			val.SetFloat(floatVal)
		}
	}
}

func getFieldValueAsString(val reflect.Value) (string, bool) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return "", false
		}

		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), true
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), true
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, val.Type().Bits()), true
	case reflect.String:
		return val.String(), true
	}

	return "", true
}

func getCookieValue(r *http.Request, name string) string {
	cookie, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setBodyValue(val reflect.Value, fieldType reflect.StructField, body io.ReadCloser, bodyType string) io.ReadCloser {
	switch bodyType {
	case "text":
		bodyBytes, err := io.ReadAll(body)
		if err == nil {
			body.Close()
			setFieldValue(val, fieldType, string(bodyBytes))
			return io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	case "json":
		bodyBytes, err := io.ReadAll(body)
		if err == nil {
			body.Close()
			err = json.Unmarshal(bodyBytes, val.Addr().Interface())
			return io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
		// TODO: handle x-www-form-urlencoded
		// TODO: handle multipart/form-data
	}

	return body
}

func getResponseBody(val reflect.Value, bodyType string) ([]byte, string) {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil, ""
		}

		val = val.Elem()
	}

	switch bodyType {
	case "text":
		valueAsString, _ := getFieldValueAsString(val)
		return []byte(valueAsString), "text/plain; charset=utf-8"
	case "json":
		body, err := json.Marshal(val.Interface())
		if err == nil {
			return body, "application/json; charset=utf-8"
		}
	}
	// TODO: add support for xml
	// TODO: add support for custom marshalers

	return nil, ""
}

// getCookiesFromFieldVal converts a response field value into an http.Cookie.
//
// Extracts cookie data from handler return values tagged with "cookie" bind tag.
// Supports string (creates basic cookie with given name) and http.Cookie (preserves all attributes).
// Automatically dereferences pointers; nil pointers and unsupported types return nil.
//
// Parameters:
//   - val: reflect.Value containing string, http.Cookie, or pointer to either.
//   - name: Cookie name for string input; ignored for http.Cookie (uses struct's Name).
//
// Returns *http.Cookie ready for http.SetCookie, or nil if input is nil/unsupported.
func getCookiesFromFieldVal(val reflect.Value, name string) *http.Cookie {
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return nil
		}

		val = val.Elem()
	}

	switch val.Kind() {
	case reflect.String:
		return &http.Cookie{
			Name:  name,
			Value: val.String(),
		}
	case reflect.Struct:
		if val.Type() == reflect.TypeFor[http.Cookie]() {
			httpCookie := val.Interface().(http.Cookie)
			return &httpCookie
		}
	}

	return nil
}

// buildStructBinding inspects a struct type for HTTP binding tags and returns a StructBinding.
//
// It examines exported fields for the following tags (in priority order):
//   - header: binds to an HTTP header
//   - query: binds to a URL query parameter
//   - path: binds to a path parameter
//   - body: binds to the request/response body
//   - cookie: binds to an HTTP cookie
//
// Only the first matching tag per field is used. Untagged and unexported fields are ignored.
// Fields are processed in declaration order.
func buildStructBinding(refType reflect.Type) StructBinding {
	binding := StructBinding{}

	for i := 0; i < refType.NumField(); i++ {
		field := refType.Field(i)

		header, isHeader := field.Tag.Lookup("header")
		query, isQuery := field.Tag.Lookup("query")
		path, isPath := field.Tag.Lookup("path")
		body, isBody := field.Tag.Lookup("body")
		cookie, isCookie := field.Tag.Lookup("cookie")

		switch {
		case isHeader:
			binding.Fields = append(binding.Fields, FieldBinding{Field: field, TagType: "header", TagValue: header})
		case isQuery:
			binding.Fields = append(binding.Fields, FieldBinding{Field: field, TagType: "query", TagValue: query})
		case isPath:
			binding.Fields = append(binding.Fields, FieldBinding{Field: field, TagType: "path", TagValue: path})
		case isBody:
			binding.Fields = append(binding.Fields, FieldBinding{Field: field, TagType: "body", TagValue: body})
		case isCookie:
			binding.Fields = append(binding.Fields, FieldBinding{Field: field, TagType: "cookie", TagValue: cookie})
		}
	}

	return binding
}

// writeResponse writes an HTTP response by serializing a struct value into headers,
// body, and cookies based on struct field tags.
//
// The responseValue must be a struct or pointer to a struct. Fields are processed
// according to their tags:
//
//   - "header": Field value is converted to string and added as a response header.
//     Multiple fields with the same header name are appended.
//   - "body": Field value is serialized as the response body. Only the first body
//     field is used. The tag value may specify a content type (e.g., "json", "xml").
//   - "cookie": Field value must be an *http.Cookie or []*http.Cookie, which are
//     set on the response via http.SetCookie.
//
// Headers are written first, then the status code (if non-zero), then the body.
// The binding for responseType is cached for subsequent calls.
//
// Parameters:
//   - w: The http.ResponseWriter to write the response to.
//   - _: The http.Request (unused, retained for handler interface compatibility).
//   - statusCode: HTTP status code to write. If 0, WriteHeader is not called.
//   - responseValue: Reflect value of the response struct to serialize. Must be
//     a valid struct or pointer to struct.
//   - responseType: Reflect type of the response struct for binding cache lookup.
//
// Panics if responseValue is not a struct or pointer to struct, or if reflection
// operations fail due to unexported fields.
func writeResponse(
	w http.ResponseWriter,
	_ *http.Request,
	statusCode int,
	responseValue reflect.Value,
	responseType reflect.Type,
) {
	binding, ok := bindingCache.Get(responseType)
	if !ok {
		binding = buildStructBinding(responseType)
		bindingCache.Set(responseType, binding)
	}

	// Collect all headers
	headers := http.Header{}
	respBody := []byte{}

	for _, field := range binding.Fields {
		fieldVal := responseValue.FieldByIndex(field.Field.Index)

		switch field.TagType {
		case "header":
			valueAsString, ok := getFieldValueAsString(fieldVal)
			if ok {
				headers[field.TagValue] = append(headers[field.TagValue], valueAsString)
			}
		case "body":
			b, contentType := getResponseBody(fieldVal, field.TagValue)
			if b != nil {
				if contentType != "" {
					w.Header().Set("Content-Type", contentType)
				}

				respBody = b
			}
		case "cookie":
			httpCookie := getCookiesFromFieldVal(fieldVal, field.TagValue)
			if httpCookie != nil {
				http.SetCookie(w, httpCookie)
			}
		}
	}

	// Apply headers to response
	for key, values := range headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	if statusCode != 0 {
		w.WriteHeader(int(statusCode))
	}

	// Write body
	if respBody != nil {
		w.Write(respBody)
	}
}
