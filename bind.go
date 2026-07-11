package gobind

import (
	"bytes"
	"context"
	"encoding/json"
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

type Request[T any] struct {
	Request T
	Http    Http
}

type Response[T any] struct {
	StatusCode int
	Response   T
}

type HandlerFunc[Req any, Resp any] func(context.Context, *Request[Req]) (*Response[Resp], error)

type TypeField struct {
	// FieldType is the type of the field at index
	FieldType reflect.StructField

	// FieldIndex is the index of the field
	FieldIndex int

	// FieldValueType is the type of value
	FieldValueType string

	// FieldValue
	FieldValue string
}

type Type struct {
	Fields []TypeField
}

type TypeMap struct {
	m       *sync.Mutex
	typeMap map[reflect.Type]Type
}

func (rm *TypeMap) AddType(reflectType reflect.Type, t Type) {
	rm.m.Lock()
	defer rm.m.Unlock()

	rm.typeMap[reflectType] = t
}

func (rm *TypeMap) Get(reflectType reflect.Type) (Type, bool) {
	val, ok := rm.typeMap[reflectType]
	return val, ok
}

var (
	typeMap = TypeMap{
		m:       &sync.Mutex{},
		typeMap: make(map[reflect.Type]Type),
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

	requestType := Type{}

	for i := 0; i < reqType.NumField(); i++ {
		fieldType := reqType.Field(i)

		header, isHeader := fieldType.Tag.Lookup("header")
		query, isQuery := fieldType.Tag.Lookup("query")
		path, isPath := fieldType.Tag.Lookup("path")
		body, isBody := fieldType.Tag.Lookup("body")
		// TODO: Handle dedicated cookie tag

		switch {
		case isHeader:
			requestType.Fields = append(requestType.Fields, TypeField{
				FieldType:      fieldType,
				FieldIndex:     i,
				FieldValueType: "header",
				FieldValue:     header,
			})
		case isQuery:
			requestType.Fields = append(requestType.Fields, TypeField{
				FieldType:      fieldType,
				FieldIndex:     i,
				FieldValueType: "query",
				FieldValue:     query,
			})
		case isPath:
			requestType.Fields = append(requestType.Fields, TypeField{
				FieldType:      fieldType,
				FieldIndex:     i,
				FieldValueType: "path",
				FieldValue:     path,
			})
		case isBody:
			requestType.Fields = append(requestType.Fields, TypeField{
				FieldType:      fieldType,
				FieldIndex:     i,
				FieldValueType: "body",
				FieldValue:     body,
			})
		}
	}

	typeMap.AddType(reqType, requestType)

	// Response
	respType := reflect.TypeFor[Resp]()
	responseType := Type{}

	for i := 0; i < respType.NumField(); i++ {
		fieldType := respType.Field(i)

		header, isHeader := fieldType.Tag.Lookup("header")
		body, isBody := fieldType.Tag.Lookup("body")
		cookie, isCookie := fieldType.Tag.Lookup("cookie")

		switch {
		case isHeader:
			responseType.Fields = append(responseType.Fields, TypeField{
				FieldIndex:     i,
				FieldType:      fieldType,
				FieldValueType: "header",
				FieldValue:     header,
			})
		case isBody:
			responseType.Fields = append(responseType.Fields, TypeField{
				FieldIndex:     i,
				FieldType:      fieldType,
				FieldValueType: "body",
				FieldValue:     body,
			})
		case isCookie:
			responseType.Fields = append(responseType.Fields, TypeField{
				FieldIndex:     i,
				FieldType:      fieldType,
				FieldValueType: "cookie",
				FieldValue:     cookie,
			})
		}
	}

	typeMap.AddType(respType, responseType)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPtr := reflect.New(reqType)
		reqVal := reqPtr.Elem()

		t, ok := typeMap.Get(reqType)
		if ok {
			for _, field := range t.Fields {
				fieldVal := reqVal.Field(field.FieldIndex)

				if !fieldVal.CanSet() {
					continue
				}

				switch field.FieldValueType {
				case "header":
					setFieldValue(fieldVal, field.FieldType, r.Header.Get(field.FieldValue))
				case "query":
					setFieldValue(fieldVal, field.FieldType, r.URL.Query().Get(field.FieldValue))
				case "path":
					setFieldValue(fieldVal, field.FieldType, r.PathValue(field.FieldValue))
				case "body":
					if r.Body != nil {
						r.Body = setBodyValue(fieldVal, field.FieldType, r.Body, field.FieldValue)
					}
				}
			}
		}

		var req Req
		if isReqKindPtr {
			req = reqVal.Addr().Interface().(Req)
		} else {
			req = reqVal.Interface().(Req)
		}

		request := &Request[Req]{
			Request: req,
			Http: Http{
				R: r,
				W: w,
			},
		}

		resp, err := next(r.Context(), request)
		if err != nil {
			fmt.Println(err)
		}

		// Make sure we are getting back the exact response type.
		gotRespType := reflect.TypeOf(resp.Response)
		if gotRespType != respType {
			panic(fmt.Sprintf("expected response type of %s, but got %s", respType.Name(), gotRespType.Name()))
		}

		respVal := reflect.ValueOf(resp.Response)

		// Collect all headers
		headers := http.Header{}
		respBody := []byte{}

		responseType, ok := typeMap.Get(respType)
		if ok {
			for _, field := range responseType.Fields {
				fieldVal := respVal.Field(field.FieldIndex)

				switch field.FieldValueType {
				case "header":
					valueAsString, ok := getFieldValueAsString(fieldVal)
					if ok {
						headers[field.FieldValue] = append(headers[field.FieldValue], valueAsString)
					}
				case "body":
					b, contentType := getResponseBody(fieldVal, field.FieldValue)
					if b != nil {
						if contentType != "" {
							w.Header().Set("Content-Type", contentType)
						}

						respBody = b
					}
				case "cookie":
					httpCookie := getCookiesFromFieldVal(fieldVal, field.FieldValue)
					if httpCookie != nil {
						http.SetCookie(w, httpCookie)
					}
				}
			}
		}

		// Apply headers to response
		for key, values := range headers {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		if resp.StatusCode != 0 {
			w.WriteHeader(resp.StatusCode)
		}

		// Write body
		if respBody != nil {
			w.Write(respBody)
		}
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

// getCookiesFromFieldVal returns a cookie from the provided reflect.Value,
// Only supports string and struct(of type http.Cookie) as valid types to fetch values from.
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
