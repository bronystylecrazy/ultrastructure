package analyzer

import "testing"

import "golang.org/x/tools/go/packages"

func TestAnalyze_SampleCreateBulk(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/sample"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(report.Packages))
	}
	if len(report.Packages[0].Handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(report.Packages[0].Handlers))
	}
	if len(report.Packages[0].Routes) != 1 {
		t.Fatalf("expected 1 route binding, got %d", len(report.Packages[0].Routes))
	}
	route := report.Packages[0].Routes[0]
	if route.Method != "POST" || route.Path != "/templates/{template_id}/bulk" {
		t.Fatalf("unexpected route binding: %+v", route)
	}

	h := report.Packages[0].Handlers[0]
	if h.Name != "CreateBulk" {
		t.Fatalf("expected handler CreateBulk, got %s", h.Name)
	}
	if h.Request == nil || h.Request.Type != "sample.CreateBulkRequest" {
		t.Fatalf("expected request type sample.CreateBulkRequest, got %+v", h.Request)
	}
	if h.Request.Confidence != "inferred" {
		t.Fatalf("expected inferred request confidence for req variable, got %+v", h.Request)
	}
	if len(h.Request.ContentTypes) == 0 || h.Request.ContentTypes[0] != "application/json" {
		t.Fatalf("expected json request content type, got %+v", h.Request.ContentTypes)
	}

	foundTemplateID := false
	for _, p := range h.Path {
		if p.Name == "template_id" && p.Type == "integer" && p.Confidence == "exact" {
			foundTemplateID = true
		}
	}
	if !foundTemplateID {
		t.Fatalf("expected template_id path parameter with integer type, got %+v", h.Path)
	}

	has400 := false
	has409 := false
	has201 := false
	for _, resp := range h.Responses {
		switch resp.Status {
		case 400:
			has400 = resp.Type == "web.Error" && resp.ContentType == "application/json"
		case 409:
			has409 = resp.Type == "web.Error" && resp.ContentType == "application/json"
		case 201:
			has201 = resp.Type == "web.Response" && resp.ContentType == "application/json"
		}
	}
	if !has400 || !has409 || !has201 {
		t.Fatalf("expected responses 400/409/201 with web.Error/web.Response, got %+v", h.Responses)
	}
}

func TestSelectPackagesByScope(t *testing.T) {
	roots := []*packages.Package{
		{ID: "r1", PkgPath: "github.com/acme/proj/examples/app"},
	}
	all := []*packages.Package{
		{ID: "r1", PkgPath: "github.com/acme/proj/examples/app"},
		{ID: "w1", PkgPath: "github.com/acme/proj/internal/mod"},
		{ID: "x1", PkgPath: "github.com/other/lib"},
	}

	gotRoots := selectPackagesByScope(roots, all, "roots")
	if len(gotRoots) != 1 || gotRoots[0].PkgPath != "github.com/acme/proj/examples/app" {
		t.Fatalf("unexpected roots scope result: %+v", gotRoots)
	}

	gotWorkspace := selectPackagesByScope(roots, all, "workspace")
	if len(gotWorkspace) != 2 {
		t.Fatalf("expected 2 workspace packages, got %d (%+v)", len(gotWorkspace), gotWorkspace)
	}

	gotAll := selectPackagesByScope(roots, all, "all")
	if len(gotAll) != 3 {
		t.Fatalf("expected 3 all-scope packages, got %d (%+v)", len(gotAll), gotAll)
	}

	gotReferenced := selectPackagesByScope(roots, all, "referenced")
	if len(gotReferenced) != 3 {
		t.Fatalf("expected referenced to behave like all at package-selection stage, got %d (%+v)", len(gotReferenced), gotReferenced)
	}
}

func TestAnalyze_RouteVariants_AssignDeclAndMultiContentResponses(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/routevariants"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(report.Packages))
	}
	p := report.Packages[0]
	if len(p.Routes) != 6 {
		t.Fatalf("expected 6 route bindings, got %d (%+v)", len(p.Routes), p.Routes)
	}

	expectedRoutes := map[string]struct{}{
		"GET /v/assign":           {},
		"GET /v/decl":             {},
		"GET /v/multi":            {},
		"GET /v/ambiguous":        {},
		"GET /v/pathonly/{slug}":  {},
		"POST /v/confidence/{id}": {},
	}
	for _, route := range p.Routes {
		key := route.Method + " " + route.Path
		if _, ok := expectedRoutes[key]; !ok {
			t.Fatalf("unexpected route binding: %+v", route)
		}
		if key == "GET /v/pathonly/{slug}" {
			if route.Name != "PathOnlyRoute" {
				t.Fatalf("expected route name from @autoswag:name, got %+v", route)
			}
			if route.Description != "Path-only endpoint" {
				t.Fatalf("expected route description from @autoswag:description, got %+v", route)
			}
			if len(route.Tags) != 2 || route.Tags[0] != "pathonly" || route.Tags[1] != "demo" {
				t.Fatalf("expected route tags from @autoswag:tag, got %+v", route.Tags)
			}
		}
		delete(expectedRoutes, key)
	}
	if len(expectedRoutes) != 0 {
		t.Fatalf("missing route bindings: %+v", expectedRoutes)
	}

	var multi *HandlerReport
	for i := range p.Handlers {
		if p.Handlers[i].Name == "MultiResponse" {
			multi = &p.Handlers[i]
			break
		}
	}
	if multi == nil {
		t.Fatalf("expected MultiResponse handler report, got %+v", p.Handlers)
	}

	hasJSON := false
	hasText := false
	for _, resp := range multi.Responses {
		if resp.Status != 200 {
			continue
		}
		if resp.ContentType == "application/json" && resp.Type == "web.Response" {
			hasJSON = true
		}
		if resp.ContentType == "text/plain" && resp.Type == "string" {
			hasText = true
		}
	}
	if !hasJSON || !hasText {
		t.Fatalf("expected both text/plain and application/json for 200 response, got %+v", multi.Responses)
	}
	for _, resp := range multi.Responses {
		if resp.Status == 200 && (resp.ContentType == "application/json" || resp.ContentType == "text/plain") && resp.Confidence != "exact" {
			t.Fatalf("expected exact confidence for distinct status/content type responses, got %+v", multi.Responses)
		}
	}

	var ambiguous *HandlerReport
	for i := range p.Handlers {
		if p.Handlers[i].Name == "AmbiguousResponse" {
			ambiguous = &p.Handlers[i]
			break
		}
	}
	if ambiguous == nil {
		t.Fatalf("expected AmbiguousResponse handler report, got %+v", p.Handlers)
	}
	hasHeuristic := false
	for _, resp := range ambiguous.Responses {
		if resp.Status == 200 && resp.ContentType == "application/json" && resp.Confidence == "heuristic" {
			hasHeuristic = true
		}
	}
	if !hasHeuristic {
		t.Fatalf("expected heuristic confidence for ambiguous same-status/json responses, got %+v", ambiguous.Responses)
	}

	var pathOnly *HandlerReport
	var confidence *HandlerReport
	for i := range p.Handlers {
		if p.Handlers[i].Name == "PathOnly" {
			pathOnly = &p.Handlers[i]
		}
		if p.Handlers[i].Name == "Confidence" {
			confidence = &p.Handlers[i]
		}
	}
	if pathOnly == nil || confidence == nil {
		t.Fatalf("expected PathOnly and Confidence handlers, got %+v", p.Handlers)
	}
	if len(pathOnly.Path) != 1 || pathOnly.Path[0].Name != "slug" || pathOnly.Path[0].Type != "string" || pathOnly.Path[0].Confidence != "inferred" {
		t.Fatalf("expected inferred string slug path param, got %+v", pathOnly.Path)
	}
	if confidence.Request == nil || confidence.Request.Type != "routevariants.confidenceBody" || confidence.Request.Confidence != "inferred" {
		t.Fatalf("expected inferred confidenceBody request, got %+v", confidence.Request)
	}
	if confidence.Query == nil || confidence.Query.Type != "routevariants.confidenceQuery" || confidence.Query.Confidence != "inferred" {
		t.Fatalf("expected inferred confidenceQuery query model, got %+v", confidence.Query)
	}
	foundID := false
	for _, p := range confidence.Path {
		if p.Name == "id" && p.Type == "integer" && p.Confidence == "exact" {
			foundID = true
		}
	}
	if !foundID {
		t.Fatalf("expected exact integer id path param, got %+v", confidence.Path)
	}
}

func TestAnalyze_WrapperHelpers_CrossPackageAndInjectedMethod(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappers"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) == 0 {
		t.Fatalf("expected at least one package report")
	}

	var wrappersPkg *PackageReport
	for i := range report.Packages {
		if report.Packages[i].Path == "github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappers" {
			wrappersPkg = &report.Packages[i]
			break
		}
	}
	if wrappersPkg == nil {
		t.Fatalf("expected wrappers package in report, got %+v", report.Packages)
	}

	var wrappedFunc *HandlerReport
	var wrappedMethod *HandlerReport
	for i := range wrappersPkg.Handlers {
		switch wrappersPkg.Handlers[i].Name {
		case "WrappedFunc":
			wrappedFunc = &wrappersPkg.Handlers[i]
		case "WrappedMethod":
			wrappedMethod = &wrappersPkg.Handlers[i]
		}
	}
	if wrappedFunc == nil || wrappedMethod == nil {
		t.Fatalf("expected WrappedFunc and WrappedMethod handlers, got %+v", wrappersPkg.Handlers)
	}

	assertResponse := func(t *testing.T, h *HandlerReport, status int, typ, contentType string) {
		t.Helper()
		for _, resp := range h.Responses {
			if resp.Status == status && resp.Type == typ && resp.ContentType == contentType {
				if len(resp.Trace) == 0 {
					t.Fatalf("expected trace for %s %d %s %s, got empty", h.Name, status, typ, contentType)
				}
				return
			}
		}
		t.Fatalf("expected response %d %s %s, got %+v", status, typ, contentType, h.Responses)
	}

	assertResponse(t, wrappedFunc, 200, "web.Response", "application/json")
	assertResponse(t, wrappedMethod, 201, "web.Response", "application/json")
}

func TestAnalyze_DIInjectedInterfaceWrapper(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(report.Packages))
	}
	p := report.Packages[0]
	var wrapped *HandlerReport
	for i := range p.Handlers {
		if p.Handlers[i].Name == "Wrapped" {
			wrapped = &p.Handlers[i]
			break
		}
	}
	if wrapped == nil {
		t.Fatalf("expected Wrapped handler report, got %+v", p.Handlers)
	}

	found := false
	for _, resp := range wrapped.Responses {
		if resp.Status == 200 && resp.Type == "web.Response" && resp.ContentType == "application/json" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected DI-injected interface helper response detection, got %+v", wrapped.Responses)
	}
	if report.DependencyGraph == nil || len(report.DependencyGraph.Edges) == 0 {
		t.Fatalf("expected dependency graph edges, got %+v", report.DependencyGraph)
	}
	foundEdge := false
	for _, e := range report.DependencyGraph.Edges {
		if e.Kind == "provides_as" &&
			e.From == "type:*github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi.jsonResponder" &&
			e.To == "type:github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi.Responder" {
			foundEdge = true
			break
		}
	}
	if !foundEdge {
		t.Fatalf("expected graph provides_as edge for As[Responder], got %+v", report.DependencyGraph.Edges)
	}
}

func TestAnalyze_ExplicitOnlySkipsInterfaceHelperDispatch(t *testing.T) {
	report, err := Analyze(Options{
		Dir:          ".",
		Patterns:     []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi"},
		ExplicitOnly: true,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(report.Packages))
	}
	p := report.Packages[0]
	var wrapped *HandlerReport
	for i := range p.Handlers {
		if p.Handlers[i].Name == "Wrapped" {
			wrapped = &p.Handlers[i]
			break
		}
	}
	if wrapped == nil {
		t.Fatalf("expected Wrapped handler report, got %+v", p.Handlers)
	}
	for _, resp := range wrapped.Responses {
		if resp.Status == 200 && resp.Type == "web.Response" && resp.ContentType == "application/json" {
			t.Fatalf("did not expect interface helper response in explicit-only mode, got %+v", wrapped.Responses)
		}
	}
	foundWarn := false
	for _, d := range report.Diagnostics {
		if d.Code == "helper_response_dispatch_ambiguous" {
			foundWarn = true
			break
		}
	}
	if !foundWarn {
		t.Fatalf("expected helper_response_dispatch_ambiguous warning, got %+v", report.Diagnostics)
	}
}

func TestAnalyze_ExplicitOnlyAllowsDirectLocalHelper(t *testing.T) {
	report, err := Analyze(Options{
		Dir:          ".",
		Patterns:     []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/explicitlocal"},
		ExplicitOnly: true,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package report, got %d", len(report.Packages))
	}
	pkg := report.Packages[0]
	var get *HandlerReport
	for i := range pkg.Handlers {
		if pkg.Handlers[i].Name == "Get" {
			get = &pkg.Handlers[i]
			break
		}
	}
	if get == nil {
		t.Fatalf("expected Get handler report, got %+v", pkg.Handlers)
	}
	found := false
	for _, resp := range get.Responses {
		if resp.Status == 200 && resp.Type == "web.Response" && resp.ContentType == "application/json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected local helper response inference, got %+v", get.Responses)
	}
}

func TestAnalyze_ExplicitOnlyAllowsGenericLocalHelper(t *testing.T) {
	report, err := Analyze(Options{
		Dir:          ".",
		Patterns:     []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/explicitgeneric"},
		ExplicitOnly: true,
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package report, got %d", len(report.Packages))
	}
	pkg := report.Packages[0]
	var get *HandlerReport
	for i := range pkg.Handlers {
		if pkg.Handlers[i].Name == "Get" {
			get = &pkg.Handlers[i]
			break
		}
	}
	if get == nil {
		t.Fatalf("expected Get handler report, got %+v", pkg.Handlers)
	}
	found := false
	for _, resp := range get.Responses {
		if resp.Status == 200 && resp.Type == "web.Response" && resp.ContentType == "application/json" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected generic local helper response inference, got %+v", get.Responses)
	}
}

func TestAnalyze_DIAmbiguousInterfaceDispatchDiagnostics(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi_ambig"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Diagnostics) == 0 {
		t.Fatalf("expected diagnostics for ambiguous DI dispatch")
	}
	foundDiag := false
	for _, d := range report.Diagnostics {
		if d.Code == "di_ambiguous_interface_dispatch" {
			foundDiag = true
			if d.Severity != "warning" {
				t.Fatalf("expected warning severity, got %+v", d)
			}
			if d.File == "" || d.Line <= 0 || d.Column <= 0 {
				t.Fatalf("expected file/line/column in diagnostic, got %+v", d)
			}
			if d.LineText == "" || d.Caret == "" {
				t.Fatalf("expected line_text and caret in diagnostic, got %+v", d)
			}
		}
	}
	if !foundDiag {
		t.Fatalf("expected ambiguous DI dispatch diagnostic, got %+v", report.Diagnostics)
	}
}

func TestAnalyze_DIUnresolvedInterfaceDispatchDiagnostics(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi_unresolved"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	foundDiag := false
	for _, d := range report.Diagnostics {
		if d.Code == "di_unresolved_interface_dispatch" {
			foundDiag = true
			if d.File == "" || d.Line <= 0 || d.Column <= 0 {
				t.Fatalf("expected file/line/column in diagnostic, got %+v", d)
			}
			if d.LineText == "" || d.Caret == "" {
				t.Fatalf("expected line_text and caret in diagnostic, got %+v", d)
			}
			if d.HandlerKey == "" {
				t.Fatalf("expected handler_key in diagnostic, got %+v", d)
			}
			if len(d.Routes) == 0 {
				t.Fatalf("expected route context in diagnostic, got %+v", d)
			}
			break
		}
	}
	if !foundDiag {
		t.Fatalf("expected unresolved DI dispatch diagnostic, got %+v", report.Diagnostics)
	}
}

func TestAnalyze_StrictDIFailsOnAmbiguousDispatch(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/wrappersdi_ambig"},
		StrictDI: true,
	})
	if err == nil {
		t.Fatalf("expected strict DI error, got nil (report=%+v)", report)
	}
}

func TestAnalyze_UsesUsNewGraphToScopeDIProviders(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/usnewscope"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(report.Packages))
	}
	p := report.Packages[0]
	var wrapped *HandlerReport
	for i := range p.Handlers {
		if p.Handlers[i].Name == "Wrapped" {
			wrapped = &p.Handlers[i]
			break
		}
	}
	if wrapped == nil {
		t.Fatalf("expected Wrapped handler, got %+v", p.Handlers)
	}
	foundResp := false
	for _, resp := range wrapped.Responses {
		if resp.Status == 200 && resp.Type == "web.Response" && resp.ContentType == "application/json" {
			foundResp = true
		}
	}
	if !foundResp {
		t.Fatalf("expected response inferred from UseResponderA wiring, got %+v", wrapped.Responses)
	}
	for _, d := range report.Diagnostics {
		if d.Code == "di_ambiguous_interface_dispatch" {
			t.Fatalf("unexpected ambiguity diagnostic when us.New scope selects a single module: %+v", report.Diagnostics)
		}
	}
}

func TestAnalyze_ExtractsResponseDescriptionFromNearbyComment(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/commentdesc"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 || len(report.Packages[0].Handlers) == 0 {
		t.Fatalf("expected handler report, got %+v", report.Packages)
	}
	inlineHandlers := []HandlerReport{}
	for i := range report.Packages[0].Handlers {
		if report.Packages[0].Handlers[i].Name == "inline" {
			inlineHandlers = append(inlineHandlers, report.Packages[0].Handlers[i])
		}
	}
	if len(inlineHandlers) == 0 {
		t.Fatalf("expected inline handler, got %+v", report.Packages[0].Handlers)
	}
	foundBlock := false
	foundInline := false
	for _, h := range inlineHandlers {
		for _, resp := range h.Responses {
			if resp.Status != 200 || resp.ContentType != "text/plain" {
				continue
			}
			switch resp.Description {
			case "This is example command":
				foundBlock = true
			case "Inline comment description":
				foundInline = true
			}
		}
	}
	if !foundBlock || !foundInline {
		t.Fatalf("expected both block and inline descriptions, got %+v", inlineHandlers)
	}
}

func TestAnalyze_UsNewBranchVarIncludesPossibleBranches(t *testing.T) {
	report, err := Analyze(Options{
		Dir:        ".",
		Patterns:   []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/usnewscope_branchvar"},
		IndexScope: "referenced",
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	foundAmbiguous := false
	for _, d := range report.Diagnostics {
		if d.Code == "di_ambiguous_interface_dispatch" {
			foundAmbiguous = true
			break
		}
	}
	if !foundAmbiguous {
		t.Fatalf("expected ambiguity diagnostic when both us.New branches are statically possible, got %+v", report.Diagnostics)
	}
}

func TestAnalyze_IgnoreFileDirectiveSkipsAllRouteAutoDetection(t *testing.T) {
	report, err := Analyze(Options{
		Dir:      ".",
		Patterns: []string{"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer/testdata/ignorefile"},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(report.Packages) != 1 {
		t.Fatalf("expected 1 package, got %d", len(report.Packages))
	}
	p := report.Packages[0]
	if len(p.Routes) != 0 {
		t.Fatalf("expected no routes when @autoswag:ignore-file is present, got %+v", p.Routes)
	}
}
