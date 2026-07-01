package gobind

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strconv"
)

type Request[T any] struct {
	Request            T
	HttpRequest        *http.Request
	HttpResponseWriter http.ResponseWriter
}

type Response[T any] struct {
	Response T
}

type HandlerFunc[Req any, Res any] func(context.Context, *Request[Req]) (*Response[Res], error)

func Handler[Req any, Res any](next HandlerFunc[Req, Res]) http.Handler {
	var isReqKindPtr bool
	reqType := reflect.TypeFor[Req]()
	if reqType.Kind() == reflect.Pointer {
		isReqKindPtr = true
		reqType = reqType.Elem()
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqPtr := reflect.New(reqType)
		reqVal := reqPtr.Elem()

		for i := 0; i < reqType.NumField(); i++ {
			fieldType := reqType.Field(i)
			fieldVal := reqVal.Field(i)

			if !fieldVal.CanSet() {
				continue
			}

			header, isHeader := fieldType.Tag.Lookup("header")
			query, isQuery := fieldType.Tag.Lookup("query")
			path, isPath := fieldType.Tag.Lookup("path")
			body, isBody := fieldType.Tag.Lookup("body")

			switch {
			case isHeader:
				setFieldValue(fieldVal, fieldType, r.Header.Get(header))
			case isQuery:
				setFieldValue(fieldVal, fieldType, r.URL.Query().Get(query))
			case isPath:
				setFieldValue(fieldVal, fieldType, r.PathValue(path))
			case isBody:
				if r.Body != nil {
					r.Body = setBodyValue(fieldVal, fieldType, r.Body, body)
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
			Request:            req,
			HttpRequest:        r,
			HttpResponseWriter: w,
		}

		_, err := next(r.Context(), request)
		if err != nil {
			fmt.Println(err)
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
	case reflect.Int:
		intVal, err := strconv.Atoi(value)
		if err == nil {
			val.SetInt(int64(intVal))
		} else {
			slog.Warn(value + " is not int")
		}
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, val.Type().Bits())
		if err == nil {
			val.SetInt(intVal)
		}
	case reflect.Uint:
		uintVal, err := strconv.ParseUint(value, 10, 0)
		if err == nil {
			val.SetUint(uintVal)
		}
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
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
	}

	return body
}
