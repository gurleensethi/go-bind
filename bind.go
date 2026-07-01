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

func Handler[Req any, Res any](fn HandlerFunc[Req, Res]) http.Handler {
	// Sample http request //
	httpReq, _ := http.NewRequest(http.MethodGet, "http://google.com", bytes.NewBufferString(`{ "message": "hello world!" }`))
	httpReq.Header.Set("Authorization", "Bearer my_secret_token")
	httpReq.SetPathValue("albumID", "123")
	query := httpReq.URL.Query()
	query.Set("page_size", "12")
	query.Set("q", "search query")
	query.Set("include_metadata", "true")
	httpReq.URL.RawQuery = query.Encode()
	// Sample http request //

	reqType := reflect.TypeFor[Req]()
	resType := reflect.TypeFor[Res]()
	fmt.Println("ReqType:", reqType)
	fmt.Println("ResType:", resType)

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
			fmt.Println(fieldType.Name, header)
			setFieldValue(fieldVal, fieldType, httpReq.Header.Get(header))
		case isQuery:
			fmt.Println(fieldType.Name, query)
			setFieldValue(fieldVal, fieldType, httpReq.URL.Query().Get(query))
		case isPath:
			fmt.Println(fieldType.Name, path)
			setFieldValue(fieldVal, fieldType, httpReq.PathValue(path))
		case isBody:
			fmt.Println(fieldType.Name, body)
			if httpReq.ContentLength > 0 {
				httpReq.Body = setBodyValue(fieldVal, fieldType, httpReq.Body, body)
			}
		}
	}

	fmt.Printf("%+v\n", reqVal)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

func setFieldValue(val reflect.Value, valType reflect.StructField, value string) {
	// Empty values are ignored.
	// Non-pointer values will have default zero values.
	// Pointer values will stay nil.
	if value == "" {
		return
	}

	if val.Kind() == reflect.Pointer {
		switch val.Type().Elem().Kind() {
		case reflect.String:
			val.Set(reflect.ValueOf(&value))
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(value)
			if err == nil {
				val.Set(reflect.ValueOf(&boolVal))
			}
		case reflect.Int8:
			intVal, err := strconv.ParseInt(value, 10, 8)
			if err == nil {
				i := int8(intVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Int16:
			intVal, err := strconv.ParseInt(value, 10, 16)
			if err == nil {
				i := int16(intVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Int32:
			intVal, err := strconv.ParseInt(value, 10, 32)
			if err == nil {
				i := int32(intVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Int:
			intVal, err := strconv.Atoi(value)
			if err == nil {
				val.Set(reflect.ValueOf(&intVal))
			} else {
				slog.Warn(value + " is not int")
			}
		case reflect.Uint8:
			uintVal, err := strconv.ParseUint(value, 10, 8)
			if err == nil {
				i := uint8(uintVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Uint16:
			uintVal, err := strconv.ParseUint(value, 10, 16)
			if err == nil {
				i := uint16(uintVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Uint32:
			uintVal, err := strconv.ParseUint(value, 10, 32)
			if err == nil {
				i := uint32(uintVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Uint:
			uintVal, err := strconv.ParseUint(value, 10, 0)
			if err == nil {
				i := uint(uintVal)
				val.Set(reflect.ValueOf(&i))
			}
		case reflect.Uint64:
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err == nil {
				val.Set(reflect.ValueOf(&uintVal))
			}
		case reflect.Float32:
			floatVal, err := strconv.ParseFloat(value, 32)
			if err == nil {
				f := float32(floatVal)
				val.Set(reflect.ValueOf(&f))
			}
		case reflect.Float64:
			floatVal, err := strconv.ParseFloat(value, 64)
			if err == nil {
				val.Set(reflect.ValueOf(&floatVal))
			}
		}
	} else {
		switch val.Kind() {
		case reflect.String:
			val.SetString(value)
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(value)
			if err == nil {
				val.SetBool(boolVal)
			}
		case reflect.Int8:
			intVal, err := strconv.ParseInt(value, 10, 8)
			if err == nil {
				val.Set(reflect.ValueOf(int16(intVal)))
			}
		case reflect.Int16:
			intVal, err := strconv.ParseInt(value, 10, 16)
			if err == nil {
				val.Set(reflect.ValueOf(int16(intVal)))
			}
		case reflect.Int32:
			intVal, err := strconv.ParseInt(value, 10, 32)
			if err == nil {
				val.Set(reflect.ValueOf(int32(intVal)))
			}
		case reflect.Int:
			intVal, err := strconv.Atoi(value)
			if err == nil {
				val.SetInt(int64(intVal))
			} else {
				slog.Warn(value + " is not int")
			}
		case reflect.Uint8:
			uintVal, err := strconv.ParseUint(value, 10, 8)
			if err == nil {
				val.SetUint(uintVal)
			}
		case reflect.Uint16:
			uintVal, err := strconv.ParseUint(value, 10, 16)
			if err == nil {
				val.SetUint(uintVal)
			}
		case reflect.Uint32:
			uintVal, err := strconv.ParseUint(value, 10, 32)
			if err == nil {
				val.SetUint(uintVal)
			}
		case reflect.Uint:
			uintVal, err := strconv.ParseUint(value, 10, 0)
			if err == nil {
				val.SetUint(uintVal)
			}
		case reflect.Uint64:
			uintVal, err := strconv.ParseUint(value, 10, 64)
			if err == nil {
				val.SetUint(uintVal)
			}
		case reflect.Float32:
			floatVal, err := strconv.ParseFloat(value, 32)
			if err == nil {
				val.SetFloat(floatVal)
			}
		case reflect.Float64:
			floatVal, err := strconv.ParseFloat(value, 64)
			if err == nil {
				val.SetFloat(floatVal)
			}
		}
	}
}

func setBodyValue(val reflect.Value, valType reflect.StructField, body io.ReadCloser, bodyType string) io.ReadCloser {
	switch bodyType {
	case "text":
		bodyBytes, err := io.ReadAll(body)
		if err == nil {
			body.Close()

			if val.Kind() == reflect.Pointer {
				switch val.Type().Elem().Kind() {
				case reflect.String:
					val.Set(reflect.ValueOf(string(bodyBytes)))
				}
			} else {
				switch val.Kind() {
				case reflect.String:
					val.SetString(string(bodyBytes))
				}
			}

			return io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	case "json":
		fmt.Println(val.Kind())
		bodyBytes, err := io.ReadAll(body)
		if err == nil {
			body.Close()

			if val.Kind() == reflect.Pointer {
				switch val.Type().Elem().Kind() {
				case reflect.Struct, reflect.Slice, reflect.Map:
					err = json.Unmarshal(bodyBytes, val.Addr().Interface())
				}
			} else {
				switch val.Kind() {
				case reflect.Struct, reflect.Slice, reflect.Map:
					err = json.Unmarshal(bodyBytes, val.Addr().Interface())
				}
			}

			return io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	return body
}
