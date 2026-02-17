package web

import (
	"database/sql"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/bronystylecrazy/ultrastructure/web/testtypes/pkgalpha"
	"github.com/bronystylecrazy/ultrastructure/web/testtypes/pkgbeta"
)

type patchUserSchema struct {
	Name            *string `json:"name,omitempty"`
	Email           *string `json:"email"`
	Role            string  `json:"role" validate:"required"`
	ForceRequired   *int    `json:"force_required,omitempty" validate:"required"`
	ImplicitDefault int     `json:"implicit_default"`
}

type validationTagsSchema struct {
	Username string   `json:"username" validate:"min=3,max=20"`
	Email    string   `json:"email" validate:"email"`
	Website  string   `json:"website" validate:"url"`
	Role     string   `json:"role" validate:"oneof=admin user viewer"`
	Age      int      `json:"age" validate:"gte=18,lte=99"`
	Score    float64  `json:"score" validate:"gt=0,lt=100"`
	Status   int      `json:"status" validate:"oneof=1 2 3"`
	Tags     []string `json:"tags" validate:"min=1,max=5"`
	Code     string   `json:"code" validate:"len=6"`
}

type tagValuesSchema struct {
	Limit  int    `json:"limit" default:"10" example:"25"`
	Active bool   `json:"active" default:"false" example:"true"`
	Mode   string `json:"mode" default:"basic" example:"advanced"`
}

type userStatus string

const (
	statusActive  userStatus = "active"
	statusPending userStatus = "pending"
)

type priorityLevel int

const (
	priorityLow  priorityLevel = 1
	priorityHigh priorityLevel = 5
)

type enumBackedSchema struct {
	Status   userStatus    `json:"status"`
	Priority priorityLevel `json:"priority"`
}

type timeBackedSchema struct {
	CreatedAt time.Time `json:"created_at"`
}

type timestampTime struct {
	time.Time
}

type swaggerTypeOverrideSchema struct {
	ID           sql.NullInt64  `json:"id" swaggertype:"integer"`
	RegisterTime timestampTime  `json:"register_time" swaggertype:"primitive,integer"`
	Coeffs       []big.Float    `json:"coeffs" swaggertype:"array,number"`
	Raw          timestampTime  `json:"raw"`
	OptionalID   *sql.NullInt64 `json:"optional_id,omitempty" swaggertype:"integer"`
}

type swaggerIgnoreSchema struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Ignored int    `json:"ignored" swaggerignore:"true"`
}

type replaceAndSkipSchema struct {
	ID       sql.NullInt64  `json:"id"`
	Nickname sql.NullString `json:"nickname"`
	AltID    *sql.NullInt64 `json:"alt_id,omitempty"`
}

type extensionsSchema struct {
	ID      string `json:"id" extensions:"x-nullable,x-abc=def,!x-omitempty,notx=1"`
	Count   int    `json:"count" extensions:"x-count=10"`
	Enabled bool   `json:"enabled" extensions:"x-enabled=false"`
}

type schemaNameRegistryUser struct {
	ID string `json:"id"`
}

func TestSchemaExtractor_PointerAndValidateRequiredRules(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(patchUserSchema{}))

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatalf("expected required field list")
	}

	requiredSet := map[string]bool{}
	for _, v := range required {
		requiredSet[v] = true
	}

	if requiredSet["name"] {
		t.Fatalf("expected name to be optional (pointer + omitempty)")
	}
	if requiredSet["email"] {
		t.Fatalf("expected email to be optional (pointer)")
	}
	if !requiredSet["role"] {
		t.Fatalf("expected role to be required (validate tag)")
	}
	if !requiredSet["force_required"] {
		t.Fatalf("expected force_required to be required (validate tag precedence)")
	}
	if !requiredSet["implicit_default"] {
		t.Fatalf("expected implicit_default to be required by default")
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties object")
	}

	emailSchema := properties["email"].(map[string]interface{})
	if emailSchema["type"] != "string" {
		t.Fatalf("expected pointer string field to map to string type, got %v", emailSchema["type"])
	}
	if emailSchema["nullable"] != true {
		t.Fatalf("expected pointer field to set nullable=true")
	}

	forceRequiredSchema := properties["force_required"].(map[string]interface{})
	if forceRequiredSchema["type"] != "integer" {
		t.Fatalf("expected pointer int field to map to integer type, got %v", forceRequiredSchema["type"])
	}
}

func TestSchemaExtractor_ValidationTagsToSchemaConstraints(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(validationTagsSchema{}))

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties object")
	}

	username := properties["username"].(map[string]interface{})
	if username["minLength"] != float64(3) || username["maxLength"] != float64(20) {
		t.Fatalf("expected username min/max length, got %v", username)
	}

	email := properties["email"].(map[string]interface{})
	if email["format"] != "email" {
		t.Fatalf("expected email format=email, got %v", email["format"])
	}

	website := properties["website"].(map[string]interface{})
	if website["format"] != "uri" {
		t.Fatalf("expected website format=uri, got %v", website["format"])
	}

	role := properties["role"].(map[string]interface{})
	roleEnum := role["enum"].([]interface{})
	if len(roleEnum) != 3 || roleEnum[0] != "admin" {
		t.Fatalf("expected role enum values, got %v", roleEnum)
	}

	age := properties["age"].(map[string]interface{})
	if age["minimum"] != float64(18) || age["maximum"] != float64(99) {
		t.Fatalf("expected age minimum/maximum, got %v", age)
	}

	score := properties["score"].(map[string]interface{})
	if score["exclusiveMinimum"] != true || score["exclusiveMaximum"] != true {
		t.Fatalf("expected score exclusive bounds, got %v", score)
	}

	status := properties["status"].(map[string]interface{})
	statusEnum := status["enum"].([]interface{})
	if len(statusEnum) != 3 || statusEnum[0] != int64(1) {
		t.Fatalf("expected numeric status enum values, got %v", statusEnum)
	}

	tags := properties["tags"].(map[string]interface{})
	if tags["minItems"] != float64(1) || tags["maxItems"] != float64(5) {
		t.Fatalf("expected tags min/max items, got %v", tags)
	}

	code := properties["code"].(map[string]interface{})
	if code["minLength"] != float64(6) || code["maxLength"] != float64(6) {
		t.Fatalf("expected code fixed length, got %v", code)
	}
}

func TestSchemaExtractor_AppliesExampleAndDefaultTags(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(tagValuesSchema{}))

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties object")
	}

	limit := properties["limit"].(map[string]interface{})
	if limit["default"] != int64(10) || limit["example"] != int64(25) {
		t.Fatalf("unexpected limit default/example: %v", limit)
	}

	active := properties["active"].(map[string]interface{})
	if active["default"] != false || active["example"] != true {
		t.Fatalf("unexpected active default/example: %v", active)
	}

	mode := properties["mode"].(map[string]interface{})
	if mode["default"] != "basic" || mode["example"] != "advanced" {
		t.Fatalf("unexpected mode default/example: %v", mode)
	}

	objExample, ok := schema["example"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected object example")
	}
	if objExample["limit"] != int64(25) {
		t.Fatalf("expected object example to use explicit tag for limit, got %v", objExample["limit"])
	}
}

func TestSchemaExtractor_AppliesRegisteredEnumsForCustomTypes(t *testing.T) {
	ClearEnumRegistry()
	defer ClearEnumRegistry()

	RegisterEnum(statusActive, statusPending)
	RegisterEnum(priorityLow, priorityHigh)

	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(enumBackedSchema{}))
	properties := schema["properties"].(map[string]interface{})

	statusSchema := properties["status"].(map[string]interface{})
	statusEnum := statusSchema["enum"].([]interface{})
	if len(statusEnum) != 2 || statusEnum[0] != statusActive || statusEnum[1] != statusPending {
		t.Fatalf("unexpected status enum: %v", statusEnum)
	}

	prioritySchema := properties["priority"].(map[string]interface{})
	priorityEnum := prioritySchema["enum"].([]interface{})
	if len(priorityEnum) != 2 || priorityEnum[0] != priorityLow || priorityEnum[1] != priorityHigh {
		t.Fatalf("unexpected priority enum: %v", priorityEnum)
	}
}

func TestSchemaExtractor_TimeFieldUsesDateTimeString(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(timeBackedSchema{}))
	properties := schema["properties"].(map[string]interface{})

	createdAtSchema := properties["created_at"].(map[string]interface{})
	if createdAtSchema["type"] != "string" {
		t.Fatalf("expected created_at type=string, got %v", createdAtSchema["type"])
	}
	if createdAtSchema["format"] != "date-time" {
		t.Fatalf("expected created_at format=date-time, got %v", createdAtSchema["format"])
	}
}

func TestSchemaExtractor_AppliesSwaggerTypeOverrides(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(swaggerTypeOverrideSchema{}))
	properties := schema["properties"].(map[string]interface{})

	idSchema := properties["id"].(map[string]interface{})
	if idSchema["type"] != "integer" {
		t.Fatalf("expected id override type=integer, got %v", idSchema["type"])
	}

	registerTimeSchema := properties["register_time"].(map[string]interface{})
	if registerTimeSchema["type"] != "integer" {
		t.Fatalf("expected register_time override type=integer, got %v", registerTimeSchema["type"])
	}

	coeffsSchema := properties["coeffs"].(map[string]interface{})
	if coeffsSchema["type"] != "array" {
		t.Fatalf("expected coeffs override type=array, got %v", coeffsSchema["type"])
	}
	items := coeffsSchema["items"].(map[string]interface{})
	if items["type"] != "number" {
		t.Fatalf("expected coeffs override items type=number, got %v", items["type"])
	}

	rawSchema := properties["raw"].(map[string]interface{})
	if rawSchema["type"] != "object" {
		t.Fatalf("expected raw without override to remain object, got %v", rawSchema["type"])
	}

	optionalIDSchema := properties["optional_id"].(map[string]interface{})
	if optionalIDSchema["type"] != "integer" {
		t.Fatalf("expected optional_id override type=integer, got %v", optionalIDSchema["type"])
	}
	if optionalIDSchema["nullable"] != true {
		t.Fatalf("expected optional_id nullable=true")
	}
}

func TestSchemaExtractor_SwaggerIgnoreExcludesField(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(swaggerIgnoreSchema{}))
	properties := schema["properties"].(map[string]interface{})

	if _, exists := properties["ignored"]; exists {
		t.Fatalf("expected ignored field to be excluded")
	}
	if _, exists := properties["id"]; !exists {
		t.Fatalf("expected id field to be included")
	}
}

func TestSchemaExtractor_AppliesExtensionsTag(t *testing.T) {
	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(extensionsSchema{}))
	properties := schema["properties"].(map[string]interface{})
	idSchema := properties["id"].(map[string]interface{})

	if idSchema["x-nullable"] != true {
		t.Fatalf("expected x-nullable=true, got %v", idSchema["x-nullable"])
	}
	if idSchema["x-abc"] != "def" {
		t.Fatalf("expected x-abc=def, got %v", idSchema["x-abc"])
	}
	if idSchema["x-omitempty"] != false {
		t.Fatalf("expected x-omitempty=false, got %v", idSchema["x-omitempty"])
	}
	if _, exists := idSchema["notx"]; exists {
		t.Fatalf("expected non x-* extension to be ignored")
	}

	countSchema := properties["count"].(map[string]interface{})
	if countSchema["x-count"] != int64(10) {
		t.Fatalf("expected x-count=10 as int64, got %T(%v)", countSchema["x-count"], countSchema["x-count"])
	}

	enabledSchema := properties["enabled"].(map[string]interface{})
	if enabledSchema["x-enabled"] != false {
		t.Fatalf("expected x-enabled=false, got %v", enabledSchema["x-enabled"])
	}
}

func TestSchemaExtractor_UsesRegisteredSchemaName(t *testing.T) {
	ClearSchemaNameRegistry()
	defer ClearSchemaNameRegistry()

	RegisterSchemaName[schemaNameRegistryUser]("UserModel")

	extractor := NewSchemaExtractor()
	ref := extractor.ExtractSchemaRef(reflect.TypeOf(schemaNameRegistryUser{}))
	if ref["$ref"] != "#/components/schemas/UserModel" {
		t.Fatalf("expected $ref to use registered name, got %v", ref["$ref"])
	}

	schemas := extractor.GetSchemas()
	if _, ok := schemas["UserModel"]; !ok {
		t.Fatalf("expected component schema key UserModel to be registered")
	}
}

func TestSchemaExtractor_DeduplicatesSameTypeNameAcrossPackages(t *testing.T) {
	extractor := NewSchemaExtractor()

	refAlpha := extractor.ExtractSchemaRef(reflect.TypeOf(pkgalpha.Duplicate{}))
	refBeta := extractor.ExtractSchemaRef(reflect.TypeOf(pkgbeta.Duplicate{}))

	alphaRef, _ := refAlpha["$ref"].(string)
	betaRef, _ := refBeta["$ref"].(string)
	if alphaRef == "" || betaRef == "" {
		t.Fatalf("expected schema refs for both types, got alpha=%q beta=%q", alphaRef, betaRef)
	}
	if alphaRef == betaRef {
		t.Fatalf("expected distinct refs for same type name across packages, got %q", alphaRef)
	}

	schemas := extractor.GetSchemas()
	if _, ok := schemas["Duplicate"]; !ok {
		t.Fatalf("expected first schema key Duplicate")
	}
	if _, ok := schemas["pkgbeta_Duplicate"]; !ok {
		t.Fatalf("expected deduplicated key pkgbeta_Duplicate, got keys=%v", reflect.ValueOf(schemas).MapKeys())
	}
}

func TestSchemaExtractor_AppliesTypeReplaceAndSkipRules(t *testing.T) {
	ClearTypeRules()
	defer ClearTypeRules()

	ReplaceType[sql.NullInt64, int64]()
	SkipType[sql.NullString]()

	extractor := NewSchemaExtractor()
	schema := extractor.ExtractSchema(reflect.TypeOf(replaceAndSkipSchema{}))
	properties := schema["properties"].(map[string]interface{})

	idSchema := properties["id"].(map[string]interface{})
	if idSchema["type"] != "integer" {
		t.Fatalf("expected id replaced to integer, got %v", idSchema["type"])
	}

	altIDSchema := properties["alt_id"].(map[string]interface{})
	if altIDSchema["type"] != "integer" {
		t.Fatalf("expected alt_id replaced to integer, got %v", altIDSchema["type"])
	}
	if altIDSchema["nullable"] != true {
		t.Fatalf("expected alt_id nullable=true")
	}

	if _, exists := properties["nickname"]; exists {
		t.Fatalf("expected nickname field to be skipped")
	}
}
