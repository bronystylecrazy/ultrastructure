package web

import "reflect"

// Summary returns a RouteOption that sets operation summary.
func Summary(summary string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Summary(summary)
	}
}

// Description returns a RouteOption that sets operation description.
func Description(description string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Description(description)
	}
}

// Body returns a RouteOption that sets the request body schema.
func Body(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Body(requestType)
	}
}

// BodyOptional returns a RouteOption that sets an optional request body schema.
func BodyOptional(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.BodyOptional(requestType)
	}
}

// Form returns a RouteOption that sets required x-www-form-urlencoded body schema.
func Form(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Form(requestType)
	}
}

// FormOptional returns a RouteOption that sets optional x-www-form-urlencoded body schema.
func FormOptional(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.FormOptional(requestType)
	}
}

// Multipart returns a RouteOption that sets required multipart/form-data body schema.
func Multipart(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Multipart(requestType)
	}
}

// MultipartOptional returns a RouteOption that sets optional multipart/form-data body schema.
func MultipartOptional(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.MultipartOptional(requestType)
	}
}

// BodyAtLeastOne returns a RouteOption for PATCH-like payloads where at least one field must be provided.
func BodyAtLeastOne(requestType any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.BodyAtLeastOne(requestType)
	}
}

// Query returns a RouteOption that sets query parameter schema from type T.
func Query[T any]() RouteOption {
	var zero T
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Query(zero)
	}
}

// Header returns a RouteOption that adds an optional request header parameter.
func Header(name string, valueType any, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Header(name, valueType, description...)
	}
}

// HeaderRequired returns a RouteOption that adds a required request header parameter.
func HeaderRequired(name string, valueType any, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.HeaderRequired(name, valueType, description...)
	}
}

// HeaderExt returns a RouteOption that adds an optional request header parameter with extensions.
func HeaderExt(name string, valueType any, extensions string, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.HeaderExt(name, valueType, extensions, description...)
	}
}

// HeaderRequiredExt returns a RouteOption that adds a required request header parameter with extensions.
func HeaderRequiredExt(name string, valueType any, extensions string, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.HeaderRequiredExt(name, valueType, extensions, description...)
	}
}

// Headers returns a RouteOption that adds optional request headers from map name->type.
func Headers(values map[string]any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Headers(values)
	}
}

// HeadersRequired returns a RouteOption that adds required request headers from map name->type.
func HeadersRequired(values map[string]any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.HeadersRequired(values)
	}
}

// Cookie returns a RouteOption that adds an optional request cookie parameter.
func Cookie(name string, valueType any, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Cookie(name, valueType, description...)
	}
}

// CookieRequired returns a RouteOption that adds a required request cookie parameter.
func CookieRequired(name string, valueType any, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.CookieRequired(name, valueType, description...)
	}
}

// CookieExt returns a RouteOption that adds an optional request cookie parameter with extensions.
func CookieExt(name string, valueType any, extensions string, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.CookieExt(name, valueType, extensions, description...)
	}
}

// CookieRequiredExt returns a RouteOption that adds a required request cookie parameter with extensions.
func CookieRequiredExt(name string, valueType any, extensions string, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.CookieRequiredExt(name, valueType, extensions, description...)
	}
}

// Cookies returns a RouteOption that adds optional request cookies from map name->type.
func Cookies(values map[string]any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Cookies(values)
	}
}

// CookiesRequired returns a RouteOption that adds required request cookies from map name->type.
func CookiesRequired(values map[string]any) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.CookiesRequired(values)
	}
}

// SetHeaders returns a RouteOption that adds response header documentation.
func SetHeaders(statusCode int, name string, valueType any, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.SetHeaders(statusCode, name, valueType, description...)
	}
}

// SetCookies returns a RouteOption that adds Set-Cookie response header documentation.
func SetCookies(statusCode int, cookieName string, description ...string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.SetCookies(statusCode, cookieName, description...)
	}
}

// Produce returns a RouteOption that sets response schema type T.
func Produce[T any](statusCode ...int) RouteOption {
	code := 200
	if len(statusCode) > 0 {
		code = statusCode[0]
	}
	return func(b *RouteBuilder) *RouteBuilder {
		return b.Produces(zeroValueForType[T](), code)
	}
}

func ProduceWithDescription[T any](statusCode int, description string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.ProducesWithDescription(zeroValueForType[T](), statusCode, description)
	}
}

func ProduceAs[T any](statusCode int, contentType string) RouteOption {
	return func(b *RouteBuilder) *RouteBuilder {
		return b.ProducesAs(zeroValueForType[T](), statusCode, contentType)
	}
}

func Ok[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](200, description...)
}
func Create[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](201, description...)
}
func Accepted[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](202, description...)
}
func BadRequest[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](400, description...)
}
func Unauthorized[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](401, description...)
}
func Forbidden[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](403, description...)
}
func NotFound[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](404, description...)
}
func Conflict[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](409, description...)
}
func UnprocessableEntity[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](422, description...)
}
func InternalError[T any](description ...string) RouteOption {
	return produceStatusOptionWithDescription[T](500, description...)
}

func zeroValueForType[T any]() any {
	t := reflect.TypeFor[T]()
	if t == nil {
		var zero T
		return zero
	}
	if t.Kind() == reflect.Ptr {
		return reflect.New(t.Elem()).Interface()
	}
	return reflect.New(t).Elem().Interface()
}

func produceStatusOptionWithDescription[T any](statusCode int, description ...string) RouteOption {
	if len(description) == 0 {
		return Produce[T](statusCode)
	}
	return ProduceWithDescription[T](statusCode, description[0])
}
