package analyzer

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type GenerateOptions struct {
	PackageName string
	FuncName    string
	ExactOnly   bool
}

func GenerateHookSource(report *Report, opts GenerateOptions) (string, error) {
	if report == nil {
		return "", fmt.Errorf("report is nil")
	}
	pkgName := strings.TrimSpace(opts.PackageName)
	if pkgName == "" {
		pkgName = "autoswaggen"
	}
	funcName := strings.TrimSpace(opts.FuncName)
	if funcName == "" {
		funcName = "AutoDetectedSwaggerHook"
	}

	imports := map[string]string{
		"autoswag": "github.com/bronystylecrazy/ultrastructure/x/autoswag",
	}
	usedAliases := map[string]string{
		"autoswag": "github.com/bronystylecrazy/ultrastructure/x/autoswag",
	}
	aliasRemapByPkg := map[string]map[string]string{}

	handlerByKey := map[string]HandlerReport{}
	for _, p := range report.Packages {
		if _, ok := aliasRemapByPkg[p.Path]; !ok {
			aliasRemapByPkg[p.Path] = map[string]string{}
		}
		for _, h := range p.Handlers {
			if strings.TrimSpace(h.Key) != "" {
				handlerByKey[h.Key] = h
			}
			if h.Request != nil {
				collectTypeImports(h.Request.Type, p.Imports, imports, usedAliases, aliasRemapByPkg[p.Path], pkgName)
			}
			if h.Query != nil {
				collectTypeImports(h.Query.Type, p.Imports, imports, usedAliases, aliasRemapByPkg[p.Path], pkgName)
			}
			for _, r := range h.Responses {
				collectTypeImports(r.Type, p.Imports, imports, usedAliases, aliasRemapByPkg[p.Path], pkgName)
			}
		}
	}

	hasRouteCases := false
	needsReflect := false
	for _, p := range report.Packages {
		for _, route := range p.Routes {
			h, ok := handlerByKey[route.HandlerKey]
			if !ok {
				continue
			}
			hasRouteCases = true
			if handlerNeedsReflect(h) {
				needsReflect = true
			}
		}
	}

	hasFallbackCases := false
	for _, p := range report.Packages {
		for _, h := range p.Handlers {
			if h.Name != "" {
				hasFallbackCases = true
			}
			if handlerNeedsReflect(h) {
				needsReflect = true
			}
		}
	}
	if hasRouteCases {
		// Prefer exact method/path route bindings; only use fallback when route data is unavailable.
		hasFallbackCases = false
	}
	if needsReflect {
		imports["reflect"] = "reflect"
		usedAliases["reflect"] = "reflect"
	}
	if hasFallbackCases || hasRouteCases {
		imports["strings"] = "strings"
		usedAliases["strings"] = "strings"
	}

	var b strings.Builder
	b.WriteString("//go:build !autoswag_analyze\n")
	b.WriteString("// +build !autoswag_analyze\n\n")
	b.WriteString("package " + pkgName + "\n\n")
	b.WriteString("import (\n")

	aliases := make([]string, 0, len(imports))
	for alias := range imports {
		aliases = append(aliases, alias)
	}
	sort.Strings(aliases)
	for _, alias := range aliases {
		path := imports[alias]
		if alias == path {
			b.WriteString("\t\"" + path + "\"\n")
		} else {
			b.WriteString("\t" + alias + " \"" + path + "\"\n")
		}
	}
	b.WriteString(")\n\n")

	b.WriteString("func " + funcName + "(ctx *autoswag.Context) {\n")
	b.WriteString("\tif ctx == nil {\n\t\treturn\n\t}\n")
	b.WriteString("\tpath := strings.TrimSpace(ctx.Path)\n")
	b.WriteString("\tif path == \"\" {\n\t\tpath = \"/\"\n\t}\n")
	b.WriteString("\tif len(path) > 1 {\n\t\tpath = strings.TrimRight(path, \"/\")\n\t}\n")
	b.WriteString("\trouteKey := ctx.Method + \" \" + path\n")
	b.WriteString("\tswitch routeKey {\n")

	emittedRouteCase := 0
	for _, p := range report.Packages {
		for _, route := range p.Routes {
			h, ok := handlerByKey[route.HandlerKey]
			if !ok {
				continue
			}
			emittedRouteCase++
			normalizedPath := strings.TrimSpace(route.Path)
			if normalizedPath == "" {
				normalizedPath = "/"
			}
			if len(normalizedPath) > 1 {
				normalizedPath = strings.TrimRight(normalizedPath, "/")
			}
			key := strings.ToUpper(strings.TrimSpace(route.Method)) + " " + normalizedPath
			b.WriteString("\tcase " + strconv.Quote(key) + ":\n")
			emitHandlerMetadata(&b, h, pkgName, aliasRemapByPkg[p.Path], opts.ExactOnly)
			b.WriteString("\t\treturn\n")
		}
	}

	if emittedRouteCase == 0 && hasFallbackCases {
		b.WriteString("\tdefault:\n")
		b.WriteString("\t\thandler := strings.TrimSpace(ctx.Route.HandlerName)\n")
		b.WriteString("\t\tswitch {\n")

		for _, p := range report.Packages {
			for _, h := range p.Handlers {
				if h.Name == "" {
					continue
				}
				caseCond := fmt.Sprintf("strings.HasSuffix(handler, \".%s\") || strings.Contains(handler, \".%s-fm\")", h.Name, h.Name)
				b.WriteString("\t\tcase " + caseCond + ":\n")
				emitHandlerMetadata(&b, h, pkgName, aliasRemapByPkg[p.Path], opts.ExactOnly)
				b.WriteString("\t\t\treturn\n")
			}
		}
		b.WriteString("\t\tdefault:\n\t\t\treturn\n\t\t}\n")
		b.WriteString("\t\treturn\n")
	}

	b.WriteString("\tdefault:\n\t\treturn\n\t}\n")
	b.WriteString("}\n")
	return b.String(), nil
}

func emitHandlerMetadata(b *strings.Builder, h HandlerReport, localPkg string, aliasRemap map[string]string, exactOnly bool) {
	if h.Request != nil && h.Request.Type != "" {
		if !(exactOnly && normalizeResponseConfidence(h.Request.Confidence) != responseConfidenceExact) {
			reqExpr := typeInitExpr(h.Request.Type, localPkg, aliasRemap)
			if len(h.Request.ContentTypes) > 0 {
				cts := make([]string, 0, len(h.Request.ContentTypes))
				for _, ct := range h.Request.ContentTypes {
					cts = append(cts, strconv.Quote(ct))
				}
				b.WriteString("\t\tctx.SetRequestBody(" + reqExpr + ", true, " + strings.Join(cts, ", ") + ")\n")
			} else {
				b.WriteString("\t\tctx.SetRequestBody(" + reqExpr + ", true)\n")
			}
		}
	}
	if h.Query != nil && h.Query.Type != "" {
		if !(exactOnly && normalizeResponseConfidence(h.Query.Confidence) != responseConfidenceExact) {
			queryExpr := typeInitExpr(h.Query.Type, localPkg, aliasRemap)
			if queryExpr != "" {
				b.WriteString("\t\tctx.SetQuery(" + queryExpr + ")\n")
			}
		}
	}
	for _, p := range h.Path {
		if exactOnly && normalizeResponseConfidence(p.Confidence) != responseConfidenceExact {
			continue
		}
		t := "reflect.TypeOf(\"\")"
		switch p.Type {
		case "integer":
			t = "reflect.TypeOf(int64(0))"
		case "number":
			t = "reflect.TypeOf(float64(0))"
		case "boolean":
			t = "reflect.TypeOf(false)"
		}
		b.WriteString("\t\tctx.AddParameter(autoswag.ParameterMetadata{Name: " + strconv.Quote(p.Name) + ", In: \"path\", Type: " + t + ", Required: true})\n")
	}
	for _, r := range h.Responses {
		if exactOnly && strings.TrimSpace(r.Confidence) != "" && strings.TrimSpace(r.Confidence) != responseConfidenceExact {
			continue
		}
		modelExpr := typeInitExpr(r.Type, localPkg, aliasRemap)
		if modelExpr == "" {
			continue
		}
		description := strings.TrimSpace(r.Description)
		if description == "" {
			description = "Auto-detected"
		}
		if strings.TrimSpace(r.ContentType) != "" {
			b.WriteString("\t\tctx.SetResponseAs(" + strconv.Itoa(r.Status) + ", " + modelExpr + ", " + strconv.Quote(r.ContentType) + ", " + strconv.Quote(description) + ")\n")
		} else {
			b.WriteString("\t\tctx.SetResponse(" + strconv.Itoa(r.Status) + ", " + modelExpr + ", " + strconv.Quote(description) + ")\n")
		}
	}
}

func handlerNeedsReflect(h HandlerReport) bool {
	for _, p := range h.Path {
		switch p.Type {
		case "integer", "number", "boolean":
			return true
		}
	}
	return false
}

func collectTypeImports(typeName string, pkgImports map[string]string, imports map[string]string, usedAliases map[string]string, remap map[string]string, localPkg string) {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return
	}
	if typeName == "string" || typeName == "bool" || strings.HasPrefix(typeName, "int") || strings.HasPrefix(typeName, "uint") || strings.HasPrefix(typeName, "float") {
		return
	}
	aliasPattern := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.`)
	matches := aliasPattern.FindAllStringSubmatch(typeName, -1)
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		alias := strings.TrimSpace(m[1])
		if alias == "" || alias == localPkg {
			continue
		}
		if path, ok := pkgImports[alias]; ok && path != "" {
			finalAlias := assignImportAlias(alias, path, imports, usedAliases)
			remap[alias] = finalAlias
		}
	}
}

func assignImportAlias(preferredAlias, path string, imports map[string]string, usedAliases map[string]string) string {
	if existingAlias, ok := aliasForPath(path, usedAliases); ok {
		imports[existingAlias] = path
		return existingAlias
	}
	alias := sanitizeAlias(preferredAlias)
	if alias == "" {
		alias = "pkg"
	}
	if existingPath, ok := usedAliases[alias]; !ok || existingPath == path {
		usedAliases[alias] = path
		imports[alias] = path
		return alias
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s%d", alias, i)
		if existingPath, ok := usedAliases[candidate]; !ok || existingPath == path {
			usedAliases[candidate] = path
			imports[candidate] = path
			return candidate
		}
	}
}

func aliasForPath(path string, usedAliases map[string]string) (string, bool) {
	for alias, p := range usedAliases {
		if p == path {
			return alias, true
		}
	}
	return "", false
}

func sanitizeAlias(alias string) string {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return ""
	}
	var b strings.Builder
	for i, r := range alias {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_':
			b.WriteRune(r)
		case i > 0 && (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}

func typeInitExpr(typeName string, localPkg string, aliasRemap map[string]string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ""
	}
	switch typeName {
	case "string":
		return "\"\""
	case "bool":
		return "false"
	case "any", "interface{}", "error":
		return ""
	}
	if strings.Contains(typeName, "interface{") {
		return ""
	}
	if strings.HasPrefix(typeName, "int") || strings.HasPrefix(typeName, "uint") {
		return "0"
	}
	if strings.HasPrefix(typeName, "float") {
		return "0.0"
	}
	if strings.HasPrefix(typeName, localPkg+".") {
		typeName = strings.TrimPrefix(typeName, localPkg+".")
	}
	if len(aliasRemap) > 0 {
		typeName = rewriteTypeAliases(typeName, aliasRemap)
	}
	return typeName + "{}"
}

func rewriteTypeAliases(typeName string, aliasRemap map[string]string) string {
	aliasPattern := regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.`)
	return aliasPattern.ReplaceAllStringFunc(typeName, func(token string) string {
		alias := strings.TrimSuffix(token, ".")
		if mapped, ok := aliasRemap[alias]; ok && mapped != "" {
			return mapped + "."
		}
		return token
	})
}
