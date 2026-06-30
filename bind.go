package gobind

import (
	"bytes"
	"context"
	"fmt"
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
			// setString(fieldVal, "123")
		}
	}

	fmt.Printf("%+v\n", reqVal)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

func setFieldValue(val reflect.Value, valType reflect.StructField, value string) {
	if val.Kind() == reflect.Pointer {
		switch val.Type().Elem().Kind() {
		case reflect.String:
			val.Set(reflect.ValueOf(&value))
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(value)
			if err == nil {
				val.Set(reflect.ValueOf(&boolVal))
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
		}
	}
}
