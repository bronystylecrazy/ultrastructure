package analyzer

import (
	"crypto/sha256"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

type Options struct {
	Dir               string
	Patterns          []string
	Tags              string
	StrictDI          bool
	DisableCommentDetection bool
	DisableDirectiveDetection bool
	IndexScope        string
	LoadDeps          bool
	ExplicitOnly      bool
	ExplicitScope     string
	Overlay           map[string][]byte
	ToolVersion       string
	Progress          func(message string)
	PackageCacheLoad  func(pkgPath, fingerprint string) (*PackageCacheEntry, bool)
	PackageCacheStore func(pkgPath, fingerprint string, entry PackageCacheEntry)
}

type OptionsMutator func(*Options)

func DisableDirective() OptionsMutator {
	return func(o *Options) {
		if o != nil {
			o.DisableDirectiveDetection = true
		}
	}
}

func (o *Options) DisableDirective() {
	if o != nil {
		o.DisableDirectiveDetection = true
	}
}

type Report struct {
	Packages        []PackageReport      `json:"packages"`
	Diagnostics     []AnalyzerDiagnostic `json:"diagnostics,omitempty"`
	DependencyGraph *DependencyGraph     `json:"dependency_graph,omitempty"`
}

type PackageCacheEntry struct {
	Package     PackageReport        `json:"package"`
	Diagnostics []AnalyzerDiagnostic `json:"diagnostics,omitempty"`
}

type AnalyzerDiagnostic struct {
	Severity   string   `json:"severity"`
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Package    string   `json:"package,omitempty"`
	HandlerKey string   `json:"handler_key,omitempty"`
	Routes     []string `json:"routes,omitempty"`
	File       string   `json:"file,omitempty"`
	Line       int      `json:"line,omitempty"`
	Column     int      `json:"column,omitempty"`
	LineText   string   `json:"line_text,omitempty"`
	Caret      string   `json:"caret,omitempty"`
}

type DependencyGraph struct {
	Nodes []DependencyGraphNode `json:"nodes,omitempty"`
	Edges []DependencyGraphEdge `json:"edges,omitempty"`
}

type DependencyGraphNode struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
}

type DependencyGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind"`
}

type PackageReport struct {
	Path     string               `json:"path"`
	Imports  map[string]string    `json:"imports,omitempty"`
	Handlers []HandlerReport      `json:"handlers"`
	Routes   []RouteBindingReport `json:"routes,omitempty"`
}

type HandlerReport struct {
	Name      string               `json:"name"`
	Key       string               `json:"key,omitempty"`
	Receiver  string               `json:"receiver,omitempty"`
	Request   *RequestReport       `json:"request,omitempty"`
	Query     *TypeReport          `json:"query,omitempty"`
	Path      []PathParamReport    `json:"path,omitempty"`
	Responses []ResponseTypeReport `json:"responses,omitempty"`
}

type RouteBindingReport struct {
	Method      string               `json:"method"`
	Path        string               `json:"path"`
	HandlerKey  string               `json:"handler_key"`
	Name        string               `json:"name,omitempty"`
	Description string               `json:"description,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Responses   []ResponseTypeReport `json:"responses,omitempty"`
	PathParams  []PathParamReport    `json:"path_params,omitempty"`
	ResponseHeaders []RouteResponseHeaderReport `json:"response_headers,omitempty"`
	TagDescriptions map[string]string `json:"tag_descriptions,omitempty"`
}

type RouteResponseHeaderReport struct {
	Status      int    `json:"status"`
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

type RequestReport struct {
	Type         string   `json:"type"`
	ContentTypes []string `json:"content_types,omitempty"`
	Confidence   string   `json:"confidence,omitempty"`
}

type TypeReport struct {
	Type       string `json:"type"`
	Confidence string `json:"confidence,omitempty"`
}

type PathParamReport struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Confidence  string `json:"confidence,omitempty"`
	Description string `json:"description,omitempty"`
}

type ResponseTypeReport struct {
	Status      int      `json:"status"`
	Type        string   `json:"type"`
	ContentType string   `json:"content_type,omitempty"`
	Description string   `json:"description,omitempty"`
	Confidence  string   `json:"confidence,omitempty"`
	Trace       []string `json:"trace,omitempty"`
	Headers     map[string]ResponseHeaderReport `json:"headers,omitempty"`
}

type ResponseHeaderReport struct {
	Type        string   `json:"type,omitempty"`
	Description string   `json:"description,omitempty"`
	Confidence  string   `json:"confidence,omitempty"`
	Trace       []string `json:"trace,omitempty"`
}

type detectedResponse struct {
	status      int
	typ         string
	contentType string
	description string
	confidence  string
	trace       []string
	headers     map[string]ResponseHeaderReport
}

type detectedHeaderMutation struct {
	status int
	name   string
	header ResponseHeaderReport
}

const (
	responseConfidenceExact     = "exact"
	responseConfidenceInferred  = "inferred"
	responseConfidenceHeuristic = "heuristic"
)

type helperResolver struct {
	declByKey     map[string]helperFuncDecl
	provided      []providedBinding
	pkgPath       string
	strictDI      bool
	explicitOnly  bool
	diagnostics   []AnalyzerDiagnostic
	diagSeen      map[string]struct{}
	sourceByInfo  map[*types.Info]diagnosticSource
	handlerKey    string
	inStack       map[string]bool
	cache         map[string][]detectedResponse
	dispatchCache map[string][]string
	commentDetection bool
}

type helperFuncDecl struct {
	pkg *packages.Package
	fn  *ast.FuncDecl
}

type providedBinding struct {
	Concrete    types.Type
	Exports     []types.Type
	IncludeSelf bool
}

type diagnosticSource struct {
	pkgPath string
	fset    *token.FileSet
}

func Analyze(opts Options) (*Report, error) {
	session, err := NewSession(opts)
	if err != nil {
		return nil, err
	}
	return session.Analyze()
}

func normalizeIndexScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "workspace", "roots", "all", "referenced":
		return strings.ToLower(strings.TrimSpace(scope))
	default:
		return "all"
	}
}

func selectPackagesByScope(roots, all []*packages.Package, scope string) []*packages.Package {
	switch normalizeIndexScope(scope) {
	case "roots":
		return dedupePackageSlice(roots)
	case "workspace":
		prefixes := workspacePrefixesFromRoots(roots)
		if len(prefixes) == 0 {
			return dedupePackageSlice(roots)
		}
		out := make([]*packages.Package, 0, len(all))
		for _, p := range all {
			if p == nil {
				continue
			}
			for _, prefix := range prefixes {
				if p.PkgPath == prefix || strings.HasPrefix(p.PkgPath, prefix+"/") {
					out = append(out, p)
					break
				}
			}
		}
		return dedupePackageSlice(out)
	default:
		return dedupePackageSlice(all)
	}
}

func workspacePrefixesFromRoots(roots []*packages.Package) []string {
	set := map[string]struct{}{}
	for _, p := range roots {
		if p == nil {
			continue
		}
		prefix := modulePrefix(p.PkgPath)
		if prefix == "" {
			continue
		}
		set[prefix] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for prefix := range set {
		out = append(out, prefix)
	}
	sort.Strings(out)
	return out
}

func modulePrefix(pkgPath string) string {
	pkgPath = strings.TrimSpace(pkgPath)
	if pkgPath == "" {
		return ""
	}
	parts := strings.Split(pkgPath, "/")
	if len(parts) >= 3 {
		return strings.Join(parts[:3], "/")
	}
	return pkgPath
}

func dedupePackageSlice(in []*packages.Package) []*packages.Package {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]*packages.Package, 0, len(in))
	for _, p := range in {
		if p == nil {
			continue
		}
		key := p.ID
		if key == "" {
			key = p.PkgPath
		}
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
	}
	return out
}

func collectPackageImports(pkg *packages.Package) map[string]string {
	imports := map[string]string{}
	if pkg == nil || pkg.Types == nil {
		return imports
	}
	for _, imp := range pkg.Types.Imports() {
		if imp == nil {
			continue
		}
		imports[imp.Name()] = imp.Path()
	}
	imports[pkg.Types.Name()] = pkg.PkgPath
	return imports
}

func packageFingerprint(pkg *packages.Package, scope, tags string, strictDI bool, toolVersion string) (string, error) {
	if pkg == nil {
		return "", fmt.Errorf("nil package")
	}
	lines := []string{
		"pkg=" + strings.TrimSpace(pkg.PkgPath),
		"scope=" + strings.TrimSpace(scope),
		"tags=" + strings.TrimSpace(tags),
		fmt.Sprintf("strict_di=%t", strictDI),
		"tool_version=" + strings.TrimSpace(toolVersion),
	}
	fileSet := map[string]struct{}{}
	for _, path := range pkg.GoFiles {
		if strings.TrimSpace(path) == "" {
			continue
		}
		fileSet[path] = struct{}{}
	}
	files := make([]string, 0, len(fileSet))
	for path := range fileSet {
		files = append(files, path)
	}
	sort.Strings(files)
	for _, path := range files {
		if shouldIgnoreAutoswagGeneratedFile(path) {
			continue
		}
		stat, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		lines = append(lines, fmt.Sprintf("file=%s|%d|%d", path, stat.Size(), stat.ModTime().UnixNano()))
	}
	importPaths := make([]string, 0, len(pkg.Imports))
	for path := range pkg.Imports {
		importPaths = append(importPaths, path)
	}
	sort.Strings(importPaths)
	for _, path := range importPaths {
		lines = append(lines, "import="+path)
		imp := pkg.Imports[path]
		if imp == nil {
			continue
		}
		importFiles := make([]string, 0, len(imp.GoFiles))
		for _, f := range imp.GoFiles {
			if strings.TrimSpace(f) == "" {
				continue
			}
			importFiles = append(importFiles, f)
		}
		sort.Strings(importFiles)
		for _, f := range importFiles {
			stat, err := os.Stat(f)
			if err != nil {
				continue
			}
			lines = append(lines, fmt.Sprintf("import_file=%s|%d|%d", f, stat.Size(), stat.ModTime().UnixNano()))
		}
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return fmt.Sprintf("%x", sum[:]), nil
}

func shouldIgnoreAutoswagGeneratedFile(path string) bool {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(path)))
	if base == "" {
		return false
	}
	if base == "zz_autoswag_hook.gen.go" || base == "zz_autoswag_hook.stub_analyze.go" {
		return true
	}
	return strings.Contains(base, "autoswag_hook") && strings.HasSuffix(base, ".gen.go")
}

func extractNearbyCommentForNode(pkg *packages.Package, node ast.Node, enabled bool) string {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return ""
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return ""
	}
	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return ""
	}
	bestLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		isTrailingInline := start.IsValid() &&
			nodeEnd.IsValid() &&
			start.Line == nodePos.Line &&
			start.Column >= nodeEnd.Column
		if end.Line >= nodePos.Line && !isTrailingInline {
			continue
		}
		if !isTrailingInline && nodePos.Line-end.Line > 2 {
			continue
		}
		text := strings.TrimSpace(cg.Text())
		if text == "" {
			continue
		}
		candidateLine := end.Line
		if isTrailingInline {
			candidateLine = nodePos.Line + 1
		}
		if candidateLine > bestLine {
			bestLine = candidateLine
			bestText = normalizeCommentText(text)
		}
	}
	return bestText
}

func fileForNode(pkg *packages.Package, node ast.Node) *ast.File {
	for _, f := range pkg.Syntax {
		if f == nil {
			continue
		}
		if f.Pos() <= node.Pos() && node.End() <= f.End() {
			return f
		}
	}
	return nil
}

func normalizeCommentText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	fields := strings.Fields(s)
	return strings.TrimSpace(strings.Join(fields, " "))
}

func hasAutoswagIgnoreDirective(pkg *packages.Package, node ast.Node, enabled bool) bool {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return false
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return false
	}
	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return false
	}
	containsDirective := func(text string) bool {
		return strings.Contains(strings.ToLower(strings.TrimSpace(text)), "@autoswag:ignore")
	}
	containsFileDirective := func(text string) bool {
		normalized := strings.ToLower(strings.TrimSpace(text))
		return strings.Contains(normalized, "@autoswag:ignore-file") || strings.Contains(normalized, "@autoswag:file-ignore")
	}
	// File-level ignore.
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		if containsFileDirective(cg.Text()) {
			return true
		}
	}
	// Same-line trailing comment support.
	if nodeEnd.IsValid() {
		for _, cg := range file.Comments {
			if cg == nil {
				continue
			}
			start := pkg.Fset.Position(cg.Pos())
			if !start.IsValid() {
				continue
			}
			if start.Line == nodePos.Line && start.Column >= nodeEnd.Column && containsDirective(cg.Text()) {
				return true
			}
		}
	}
	bestEndLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() || end.Line >= nodePos.Line {
			continue
		}
		// Directive applies only to the immediately following statement.
		if nodePos.Line-end.Line > 1 {
			continue
		}
		// Ignore trailing comments from previous statements in "above" mode.
		if start.IsValid() && lineHasNonWhitespacePrefix(start.Filename, start.Line, start.Column) {
			continue
		}
		if end.Line > bestEndLine {
			bestEndLine = end.Line
			bestText = cg.Text()
		}
	}
	return containsDirective(bestText)
}

func lineHasNonWhitespacePrefix(path string, line, column int) bool {
	path = strings.TrimSpace(path)
	if path == "" || line <= 0 || column <= 1 {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(b), "\n")
	if line > len(lines) {
		return false
	}
	runes := []rune(lines[line-1])
	limit := column - 1
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := 0; i < limit; i++ {
		if !unicode.IsSpace(runes[i]) {
			return true
		}
	}
	return false
}

func autoswagDirectiveValue(pkg *packages.Package, node ast.Node, key string, enabled bool) string {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return ""
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return ""
	}
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return ""
	}
	parseValue := func(text string) string {
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			prefix := "@autoswag:" + key
			if !strings.HasPrefix(strings.ToLower(l), prefix) {
				continue
			}
			rest := strings.TrimSpace(l[len(prefix):])
			rest = strings.TrimSpace(strings.TrimLeft(rest, ":="))
			return rest
		}
		return ""
	}

	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return ""
	}
	// Same-line trailing directive.
	if nodeEnd.IsValid() {
		for _, cg := range file.Comments {
			if cg == nil {
				continue
			}
			start := pkg.Fset.Position(cg.Pos())
			if !start.IsValid() {
				continue
			}
			if start.Line == nodePos.Line && start.Column >= nodeEnd.Column {
				if v := parseValue(cg.Text()); v != "" {
					return v
				}
			}
		}
	}
	// Directive directly above this statement.
	bestEndLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() || end.Line >= nodePos.Line {
			continue
		}
		if nodePos.Line-end.Line > 1 {
			continue
		}
		if start.IsValid() && lineHasNonWhitespacePrefix(start.Filename, start.Line, start.Column) {
			continue
		}
		if end.Line > bestEndLine {
			bestEndLine = end.Line
			bestText = cg.Text()
		}
	}
	return parseValue(bestText)
}

func autoswagDirectiveValues(pkg *packages.Package, node ast.Node, key string, enabled bool) []string {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return nil
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return nil
	}
	key = strings.ToLower(strings.TrimSpace(key))
	if key == "" {
		return nil
	}
	parseValues := func(text string) []string {
		out := []string{}
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			prefix := "@autoswag:" + key
			if !strings.HasPrefix(strings.ToLower(l), prefix) {
				continue
			}
			rest := strings.TrimSpace(l[len(prefix):])
			rest = strings.TrimSpace(strings.TrimLeft(rest, ":="))
			if rest == "" {
				continue
			}
			for _, part := range strings.Split(rest, ",") {
				tag := strings.TrimSpace(part)
				if tag == "" {
					continue
				}
				out = appendUniqueString(out, tag)
			}
		}
		return out
	}

	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return nil
	}
	// Same-line trailing directive.
	if nodeEnd.IsValid() {
		for _, cg := range file.Comments {
			if cg == nil {
				continue
			}
			start := pkg.Fset.Position(cg.Pos())
			if !start.IsValid() {
				continue
			}
			if start.Line == nodePos.Line && start.Column >= nodeEnd.Column {
				if out := parseValues(cg.Text()); len(out) > 0 {
					return out
				}
			}
		}
	}
	// Directive directly above this statement.
	bestEndLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() || end.Line >= nodePos.Line {
			continue
		}
		if nodePos.Line-end.Line > 1 {
			continue
		}
		if start.IsValid() && lineHasNonWhitespacePrefix(start.Filename, start.Line, start.Column) {
			continue
		}
		if end.Line > bestEndLine {
			bestEndLine = end.Line
			bestText = cg.Text()
		}
	}
	return parseValues(bestText)
}

func autoswagDirectiveResponses(pkg *packages.Package, node ast.Node, enabled bool) []ResponseTypeReport {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return nil
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return nil
	}
	parseResponses := func(text string) []ResponseTypeReport {
		out := []ResponseTypeReport{}
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			prefix := "@autoswag:response"
			if !strings.HasPrefix(strings.ToLower(l), prefix) {
				continue
			}
			rest := strings.TrimSpace(l[len(prefix):])
			rest = strings.TrimSpace(strings.TrimLeft(rest, ":="))
			if rest == "" {
				continue
			}
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				continue
			}
			status, err := strconv.Atoi(strings.TrimSpace(fields[0]))
			if err != nil {
				continue
			}
			typ := strings.TrimSpace(fields[1])
			if typ == "" {
				continue
			}
			contentType := ""
			desc := ""
			if len(fields) >= 3 {
				if strings.Contains(fields[2], "/") {
					contentType = strings.TrimSpace(fields[2])
					if len(fields) > 3 {
						desc = strings.TrimSpace(strings.Join(fields[3:], " "))
					}
				} else {
					desc = strings.TrimSpace(strings.Join(fields[2:], " "))
				}
			}
			out = append(out, ResponseTypeReport{
				Status:      status,
				Type:        typ,
				ContentType: contentType,
				Description: desc,
				Confidence:  responseConfidenceExact,
			})
		}
		return out
	}
	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return nil
	}
	// Same-line trailing directive(s).
	if nodeEnd.IsValid() {
		for _, cg := range file.Comments {
			if cg == nil {
				continue
			}
			start := pkg.Fset.Position(cg.Pos())
			if !start.IsValid() {
				continue
			}
			if start.Line == nodePos.Line && start.Column >= nodeEnd.Column {
				if out := parseResponses(cg.Text()); len(out) > 0 {
					return out
				}
			}
		}
	}
	// Directive block directly above statement.
	bestEndLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() || end.Line >= nodePos.Line {
			continue
		}
		if nodePos.Line-end.Line > 1 {
			continue
		}
		if start.IsValid() && lineHasNonWhitespacePrefix(start.Filename, start.Line, start.Column) {
			continue
		}
		if end.Line > bestEndLine {
			bestEndLine = end.Line
			bestText = cg.Text()
		}
	}
	return parseResponses(bestText)
}

func autoswagDirectivePathParams(pkg *packages.Package, node ast.Node, enabled bool) []PathParamReport {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return nil
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return nil
	}
	parseParams := func(text string) []PathParamReport {
		out := []PathParamReport{}
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			lower := strings.ToLower(l)
			prefix := "@autoswag:param"
			legacyPrefix := "@autoswag:pathparam"
			matchPrefix := ""
			if strings.HasPrefix(lower, prefix) {
				matchPrefix = prefix
			}
			if strings.HasPrefix(lower, legacyPrefix) {
				matchPrefix = legacyPrefix
			}
			if matchPrefix == "" {
				continue
			}
			rest := strings.TrimSpace(l[len(matchPrefix):])
			rest = strings.TrimSpace(strings.TrimLeft(rest, ":="))
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				continue
			}
			name := strings.TrimSpace(fields[0])
			typ := strings.TrimSpace(fields[1])
			if name == "" || typ == "" {
				continue
			}
			desc := ""
			if len(fields) > 2 {
				desc = strings.TrimSpace(strings.Join(fields[2:], " "))
			}
			out = append(out, PathParamReport{
				Name:        name,
				Type:        typ,
				Confidence:  responseConfidenceExact,
				Description: desc,
			})
		}
		return out
	}
	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return nil
	}
	if nodeEnd.IsValid() {
		for _, cg := range file.Comments {
			if cg == nil {
				continue
			}
			start := pkg.Fset.Position(cg.Pos())
			if !start.IsValid() {
				continue
			}
			if start.Line == nodePos.Line && start.Column >= nodeEnd.Column {
				if out := parseParams(cg.Text()); len(out) > 0 {
					return out
				}
			}
		}
	}
	bestEndLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() || end.Line >= nodePos.Line {
			continue
		}
		if nodePos.Line-end.Line > 1 {
			continue
		}
		if start.IsValid() && lineHasNonWhitespacePrefix(start.Filename, start.Line, start.Column) {
			continue
		}
		if end.Line > bestEndLine {
			bestEndLine = end.Line
			bestText = cg.Text()
		}
	}
	return parseParams(bestText)
}

func autoswagDirectiveHeaders(pkg *packages.Package, node ast.Node, enabled bool) []RouteResponseHeaderReport {
	if !enabled || pkg == nil || pkg.Fset == nil || node == nil {
		return nil
	}
	file := fileForNode(pkg, node)
	if file == nil || len(file.Comments) == 0 {
		return nil
	}
	parseHeaders := func(text string) []RouteResponseHeaderReport {
		out := []RouteResponseHeaderReport{}
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			lower := strings.ToLower(l)
			prefix := "@autoswag:header"
			legacyPrefix := "@autoswag:response-header"
			matchPrefix := ""
			if strings.HasPrefix(lower, prefix) {
				matchPrefix = prefix
			}
			if strings.HasPrefix(lower, legacyPrefix) {
				matchPrefix = legacyPrefix
			}
			if matchPrefix == "" {
				continue
			}
			rest := strings.TrimSpace(l[len(matchPrefix):])
			rest = strings.TrimSpace(strings.TrimLeft(rest, ":="))
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				continue
			}
			status, err := strconv.Atoi(strings.TrimSpace(fields[0]))
			if err != nil {
				continue
			}
			name := strings.TrimSpace(fields[1])
			if name == "" {
				continue
			}
			typ := "string"
			desc := ""
			if len(fields) >= 3 {
				v := strings.TrimSpace(strings.ToLower(fields[2]))
				switch v {
				case "string", "integer", "number", "boolean":
					typ = v
					if len(fields) > 3 {
						desc = strings.TrimSpace(strings.Join(fields[3:], " "))
					}
				default:
					desc = strings.TrimSpace(strings.Join(fields[2:], " "))
				}
			}
			out = append(out, RouteResponseHeaderReport{
				Status:      status,
				Name:        name,
				Type:        typ,
				Description: desc,
			})
		}
		return out
	}

	nodePos := pkg.Fset.Position(node.Pos())
	nodeEnd := pkg.Fset.Position(node.End())
	if !nodePos.IsValid() {
		return nil
	}
	if nodeEnd.IsValid() {
		for _, cg := range file.Comments {
			if cg == nil {
				continue
			}
			start := pkg.Fset.Position(cg.Pos())
			if !start.IsValid() {
				continue
			}
			if start.Line == nodePos.Line && start.Column >= nodeEnd.Column {
				if out := parseHeaders(cg.Text()); len(out) > 0 {
					return out
				}
			}
		}
	}
	bestEndLine := -1
	bestText := ""
	for _, cg := range file.Comments {
		if cg == nil {
			continue
		}
		start := pkg.Fset.Position(cg.Pos())
		end := pkg.Fset.Position(cg.End())
		if !end.IsValid() || end.Line >= nodePos.Line {
			continue
		}
		if nodePos.Line-end.Line > 1 {
			continue
		}
		if start.IsValid() && lineHasNonWhitespacePrefix(start.Filename, start.Line, start.Column) {
			continue
		}
		if end.Line > bestEndLine {
			bestEndLine = end.Line
			bestText = cg.Text()
		}
	}
	return parseHeaders(bestText)
}

func annotateDiagnosticsRoutes(diags []AnalyzerDiagnostic, routes []RouteBindingReport) {
	if len(diags) == 0 || len(routes) == 0 {
		return
	}
	routeByHandler := map[string][]string{}
	for _, r := range routes {
		key := strings.TrimSpace(r.HandlerKey)
		if key == "" {
			continue
		}
		routeKey := strings.ToUpper(strings.TrimSpace(r.Method)) + " " + strings.TrimSpace(r.Path)
		if routeKey == "" {
			continue
		}
		routeByHandler[key] = appendUniqueString(routeByHandler[key], routeKey)
	}
	for i := range diags {
		key := strings.TrimSpace(diags[i].HandlerKey)
		if key == "" {
			continue
		}
		if routes, ok := routeByHandler[key]; ok && len(routes) > 0 {
			sort.Strings(routes)
			diags[i].Routes = append([]string(nil), routes...)
		}
	}
}

func appendUniqueString(values []string, v string) []string {
	v = strings.TrimSpace(v)
	if v == "" {
		return values
	}
	for _, existing := range values {
		if existing == v {
			return values
		}
	}
	return append(values, v)
}

func isFiberCtxHandler(fn *ast.FuncDecl, info *types.Info) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	resType := info.TypeOf(fn.Type.Results.List[0].Type)
	if resType == nil || resType.String() != "error" {
		return false
	}
	params := fn.Type.Params.List
	if len(params) == 0 {
		return false
	}
	last := params[len(params)-1]
	t := info.TypeOf(last.Type)
	return t != nil && t.String() == "github.com/gofiber/fiber/v3.Ctx"
}

func analyzeFunc(pkg *packages.Package, fn *ast.FuncDecl, resolver *helperResolver, commentDetection bool) HandlerReport {
	out := HandlerReport{Name: fn.Name.Name}
	out.Key = handlerDeclKey(pkg, fn)
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		out.Receiver = pkg.TypesInfo.TypeOf(fn.Recv.List[0].Type).String()
	}

	varTypes := map[string]types.Type{}
	paramsVars := map[string]string{}
	pathTypes := map[string]PathParamReport{}
	queryType := ""
	queryConfidence := ""
	reqType := ""
	reqConfidence := ""
	reqContentTypes := map[string]struct{}{}
	responses := []detectedResponse{}
	headerMutations := []detectedHeaderMutation{}

	withResolverHandlerContext(resolver, out.Key, func() {
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				recordAssignTypes(node, pkg.TypesInfo, varTypes)
				recordPathParamAssignments(node, pkg.TypesInfo, paramsVars, pathTypes)
				recordRequestType(node, pkg.TypesInfo, varTypes, &reqType, &reqConfidence)
				recordRequestContentType(node, reqContentTypes)
				recordQueryType(node, pkg.TypesInfo, varTypes, &queryType, &queryConfidence)
				recordResponseTypeFromAssign(node, pkg.TypesInfo, &responses, resolver)
				recordResponseHeaderMutationsFromAssign(node, pkg.TypesInfo, &headerMutations)
			case *ast.DeclStmt:
				recordDeclTypes(node, pkg.TypesInfo, varTypes)
			case *ast.ExprStmt:
				recordResponseHeaderMutationsFromExpr(node.X, pkg.TypesInfo, &headerMutations)
			case *ast.ReturnStmt:
				recordResponseTypeFromReturn(node, pkg, pkg.TypesInfo, &responses, resolver, commentDetection)
				recordResponseHeaderMutationsFromReturn(node, pkg.TypesInfo, &headerMutations)
			}
			return true
		})
	})

	if reqType != "" {
		out.Request = &RequestReport{
			Type:       reqType,
			Confidence: normalizeInferenceConfidence(reqConfidence),
		}
		if len(reqContentTypes) == 0 {
			reqContentTypes["application/json"] = struct{}{}
		}
		cts := make([]string, 0, len(reqContentTypes))
		for ct := range reqContentTypes {
			cts = append(cts, ct)
		}
		sort.Strings(cts)
		out.Request.ContentTypes = cts
	}
	if queryType != "" {
		out.Query = &TypeReport{
			Type:       queryType,
			Confidence: normalizeInferenceConfidence(queryConfidence),
		}
	}

	if len(pathTypes) > 0 {
		keys := make([]string, 0, len(pathTypes))
		for k := range pathTypes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.Path = make([]PathParamReport, 0, len(keys))
		for _, k := range keys {
			p := pathTypes[k]
			p.Confidence = normalizeInferenceConfidence(p.Confidence)
			out.Path = append(out.Path, p)
		}
	}

	if len(responses) > 0 {
		responses = applyHeaderMutations(responses, headerMutations)
		out.Responses = toResponseTypeReports(responses)
	}

	return out
}

func isRouterHandleMethod(fn *ast.FuncDecl, info *types.Info) bool {
	if fn.Name == nil || fn.Name.Name != "Handle" {
		return false
	}
	if fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return false
	}
	first := fn.Type.Params.List[0]
	t := info.TypeOf(first.Type)
	if t == nil {
		return false
	}
	return t.String() == "github.com/bronystylecrazy/ultrastructure/web.Router"
}

func extractRouteBindings(pkg *packages.Package, fn *ast.FuncDecl, resolver *helperResolver, commentDetection bool, directiveDetection bool) ([]RouteBindingReport, []HandlerReport) {
	if fn.Body == nil || fn.Type == nil || fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
		return nil, nil
	}
	prefixByVar := map[string]string{}
	groupTagsByVar := map[string][]string{}
	groupTagDescByVar := map[string]map[string]string{}
	firstParam := fn.Type.Params.List[0]
	if len(firstParam.Names) > 0 {
		prefixByVar[firstParam.Names[0].Name] = ""
		groupTagsByVar[firstParam.Names[0].Name] = nil
		groupTagDescByVar[firstParam.Names[0].Name] = nil
	}

	out := []RouteBindingReport{}
	inline := []HandlerReport{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.AssignStmt:
			for i, rhs := range node.Rhs {
				groupCall, ok := unwrapGroupCall(rhs)
				if !ok || len(groupCall.Args) == 0 {
					out, inline = appendRouteBindingFromExpr(rhs, pkg, fn, resolver, prefixByVar, groupTagsByVar, groupTagDescByVar, commentDetection, directiveDetection, out, inline)
					continue
				}
				receiverVar, ok := selectorRootIdent(groupCall.Fun)
				if !ok {
					continue
				}
				groupPath, ok := literalString(groupCall.Args[0])
				if !ok {
					continue
				}
				base := prefixByVar[receiverVar]
				baseTags := groupTagsByVar[receiverVar]
				baseTagDesc := groupTagDescByVar[receiverVar]
				mergedTags := append([]string(nil), baseTags...)
				mergedTags = appendUniqueStrings(mergedTags, groupTagsFromExpr(rhs)...)
				mergedTagDesc := copyStringMap(baseTagDesc)
				for k, v := range groupTagDescriptionsFromExpr(pkg, rhs, mergedTags, commentDetection, directiveDetection) {
					mergedTagDesc[k] = v
				}
				if i < len(node.Lhs) {
					if ident, ok := node.Lhs[i].(*ast.Ident); ok && ident.Name != "_" {
						prefixByVar[ident.Name] = joinRoutePath(base, groupPath)
						groupTagsByVar[ident.Name] = mergedTags
						groupTagDescByVar[ident.Name] = mergedTagDesc
					}
				}
			}
		case *ast.ExprStmt:
			out, inline = appendRouteBindingFromExpr(node.X, pkg, fn, resolver, prefixByVar, groupTagsByVar, groupTagDescByVar, commentDetection, directiveDetection, out, inline)
		case *ast.DeclStmt:
			gen, ok := node.Decl.(*ast.GenDecl)
			if !ok {
				return true
			}
			for _, spec := range gen.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, value := range vs.Values {
					out, inline = appendRouteBindingFromExpr(value, pkg, fn, resolver, prefixByVar, groupTagsByVar, groupTagDescByVar, commentDetection, directiveDetection, out, inline)
				}
			}
		}
		return true
	})
	return out, inline
}

func appendRouteBindingFromExpr(
	expr ast.Expr,
	pkg *packages.Package,
	owner *ast.FuncDecl,
	resolver *helperResolver,
	prefixByVar map[string]string,
	groupTagsByVar map[string][]string,
	groupTagDescByVar map[string]map[string]string,
	commentDetection bool,
	directiveDetection bool,
	out []RouteBindingReport,
	inline []HandlerReport,
) ([]RouteBindingReport, []HandlerReport) {
	if hasAutoswagIgnoreDirective(pkg, expr, directiveDetection) {
		return out, inline
	}
	routeName := autoswagDirectiveValue(pkg, expr, "name", directiveDetection)
	routeDescription := autoswagDirectiveValue(pkg, expr, "description", directiveDetection)
	routeTags := autoswagDirectiveValues(pkg, expr, "tag", directiveDetection)
	routeResponses := autoswagDirectiveResponses(pkg, expr, directiveDetection)
	routePathParams := autoswagDirectivePathParams(pkg, expr, directiveDetection)
	routeHeaders := autoswagDirectiveHeaders(pkg, expr, directiveDetection)
	routeTagDescriptions := map[string]string{}
	routeCall, method, ok := unwrapRouteCall(expr)
	if !ok || len(routeCall.Args) < 2 {
		return out, inline
	}
	receiverVar, ok := selectorRootIdent(routeCall.Fun)
	if !ok {
		return out, inline
	}
	if len(routeTags) == 0 {
		routeTags = append([]string(nil), groupTagsByVar[receiverVar]...)
	}
	for k, v := range groupTagDescByVar[receiverVar] {
		routeTagDescriptions[k] = v
	}
	rawPath, ok := literalString(routeCall.Args[0])
	if !ok {
		return out, inline
	}
	fullPath := joinRoutePath(prefixByVar[receiverVar], rawPath)
	openAPIPath := normalizeOpenAPIPathLocal(fullPath)
	handlerKey, inlineHandler := handlerKeyFromExpr(routeCall.Args[1], pkg, owner, resolver)
	if handlerKey == "" {
		return out, inline
	}
	if resolver != nil {
		if decl, ok := resolver.declByKey[handlerKey]; ok && decl.fn != nil && decl.pkg != nil {
			if strings.TrimSpace(routeName) == "" {
				routeName = autoswagDirectiveValue(decl.pkg, decl.fn, "name", directiveDetection)
			}
			if strings.TrimSpace(routeDescription) == "" {
				routeDescription = autoswagDirectiveValue(decl.pkg, decl.fn, "description", directiveDetection)
			}
			if len(routeTags) == 0 {
				routeTags = autoswagDirectiveValues(decl.pkg, decl.fn, "tag", directiveDetection)
			}
			if len(routeResponses) == 0 {
				routeResponses = autoswagDirectiveResponses(decl.pkg, decl.fn, directiveDetection)
			}
			if len(routePathParams) == 0 {
				routePathParams = autoswagDirectivePathParams(decl.pkg, decl.fn, directiveDetection)
			}
			if len(routeHeaders) == 0 {
				routeHeaders = autoswagDirectiveHeaders(decl.pkg, decl.fn, directiveDetection)
			}
		}
	}
	if inlineHandler != nil {
		inline = append(inline, *inlineHandler)
	}
	out = append(out, RouteBindingReport{
		Method:      method,
		Path:        openAPIPath,
		HandlerKey:  handlerKey,
		Name:        routeName,
		Description: routeDescription,
		Tags:        routeTags,
		Responses:   routeResponses,
		PathParams:  routePathParams,
		ResponseHeaders: routeHeaders,
		TagDescriptions: routeTagDescriptions,
	})
	return out, inline
}

func groupTagsFromExpr(expr ast.Expr) []string {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return nil
	}
	base := groupTagsFromExpr(sel.X)
	if sel.Sel.Name != "Tags" {
		return base
	}
	for _, arg := range call.Args {
		s, ok := literalString(arg)
		if !ok {
			continue
		}
		for _, tag := range strings.Split(s, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			base = appendUniqueString(base, tag)
		}
	}
	return base
}

func groupTagDescriptionsFromExpr(pkg *packages.Package, expr ast.Expr, currentTags []string, commentDetection bool, directiveDetection bool) map[string]string {
	out := map[string]string{}
	text := extractNearbyCommentForNode(pkg, expr, commentDetection || directiveDetection)
	if text == "" {
		return out
	}
	if directiveDetection && strings.Contains(strings.ToLower(text), "@autoswag:tag-description") {
		for _, line := range strings.Split(text, "\n") {
			l := strings.TrimSpace(line)
			if l == "" {
				continue
			}
			prefix := "@autoswag:tag-description"
			if !strings.HasPrefix(strings.ToLower(l), prefix) {
				continue
			}
			rest := strings.TrimSpace(l[len(prefix):])
			rest = strings.TrimSpace(strings.TrimLeft(rest, ":="))
			if rest == "" {
				continue
			}
			fields := strings.Fields(rest)
			if len(fields) == 0 {
				continue
			}
			if len(fields) >= 2 {
				tag := strings.TrimSpace(fields[0])
				desc := strings.TrimSpace(strings.Join(fields[1:], " "))
				if tag != "" && desc != "" {
					out[tag] = desc
				}
			}
		}
		return out
	}
	if commentDetection && len(currentTags) == 1 {
		out[currentTags[0]] = text
	}
	return out
}

func appendUniqueStrings(existing []string, values ...string) []string {
	out := append([]string(nil), existing...)
	for _, v := range values {
		out = appendUniqueString(out, v)
	}
	return out
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func unwrapGroupCall(expr ast.Expr) (*ast.CallExpr, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return nil, false
	}
	if sel.Sel.Name == "Group" {
		return call, true
	}
	switch sel.Sel.Name {
	case "Tags", "With", "Use":
		return unwrapGroupCall(sel.X)
	default:
		return nil, false
	}
}

func unwrapRouteCall(expr ast.Expr) (*ast.CallExpr, string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return nil, "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return nil, "", false
	}
	method := strings.ToUpper(sel.Sel.Name)
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
		return call, method, true
	}
	return unwrapRouteCall(sel.X)
}

func selectorRootIdent(expr ast.Expr) (string, bool) {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		if id, ok := v.X.(*ast.Ident); ok {
			return id.Name, true
		}
		return selectorRootIdent(v.X)
	case *ast.CallExpr:
		return selectorRootIdent(v.Fun)
	default:
		return "", false
	}
}

func handlerKeyFromExpr(expr ast.Expr, pkg *packages.Package, owner *ast.FuncDecl, resolver *helperResolver) (string, *HandlerReport) {
	switch v := expr.(type) {
	case *ast.SelectorExpr:
		if sel := pkg.TypesInfo.Selections[v]; sel != nil && sel.Obj() != nil {
			if fn, ok := sel.Obj().(*types.Func); ok {
				return fn.FullName(), nil
			}
		}
		if tv := pkg.TypesInfo.TypeOf(v); tv != nil {
			return tv.String(), nil
		}
	case *ast.Ident:
		if obj := pkg.TypesInfo.Uses[v]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				return fn.FullName(), nil
			}
		}
	case *ast.FuncLit:
		key := inlineHandlerKey(pkg, v, owner)
		report := analyzeFuncLit(pkg, v, key, resolver)
		return key, &report
	}
	return "", nil
}

func handlerDeclKey(pkg *packages.Package, fn *ast.FuncDecl) string {
	if fn == nil || fn.Name == nil {
		return ""
	}
	if obj := pkg.TypesInfo.Defs[fn.Name]; obj != nil {
		if f, ok := obj.(*types.Func); ok {
			return f.FullName()
		}
	}
	return fn.Name.Name
}

func inlineHandlerKey(pkg *packages.Package, lit *ast.FuncLit, owner *ast.FuncDecl) string {
	if lit == nil {
		return ""
	}
	pos := pkg.Fset.Position(lit.Pos())
	ownerName := ""
	if owner != nil && owner.Name != nil {
		ownerName = owner.Name.Name
	}
	return fmt.Sprintf("%s:%s:%d:%d:%s", pkg.PkgPath, filepath.Base(pos.Filename), pos.Line, pos.Column, ownerName)
}

func withResolverHandlerContext(resolver *helperResolver, handlerKey string, fn func()) {
	if fn == nil {
		return
	}
	if resolver == nil {
		fn()
		return
	}
	prev := resolver.handlerKey
	resolver.handlerKey = strings.TrimSpace(handlerKey)
	defer func() {
		resolver.handlerKey = prev
	}()
	fn()
}

func analyzeFuncLit(pkg *packages.Package, lit *ast.FuncLit, key string, resolver *helperResolver) HandlerReport {
	out := HandlerReport{
		Name: "inline",
		Key:  key,
	}

	varTypes := map[string]types.Type{}
	paramsVars := map[string]string{}
	pathTypes := map[string]PathParamReport{}
	queryType := ""
	queryConfidence := ""
	reqType := ""
	reqConfidence := ""
	reqContentTypes := map[string]struct{}{}
	responses := []detectedResponse{}
	headerMutations := []detectedHeaderMutation{}

	withResolverHandlerContext(resolver, key, func() {
		ast.Inspect(lit.Body, func(n ast.Node) bool {
			switch node := n.(type) {
			case *ast.AssignStmt:
				recordAssignTypes(node, pkg.TypesInfo, varTypes)
				recordPathParamAssignments(node, pkg.TypesInfo, paramsVars, pathTypes)
				recordRequestType(node, pkg.TypesInfo, varTypes, &reqType, &reqConfidence)
				recordRequestContentType(node, reqContentTypes)
				recordQueryType(node, pkg.TypesInfo, varTypes, &queryType, &queryConfidence)
				recordResponseTypeFromAssign(node, pkg.TypesInfo, &responses, resolver)
				recordResponseHeaderMutationsFromAssign(node, pkg.TypesInfo, &headerMutations)
			case *ast.DeclStmt:
				recordDeclTypes(node, pkg.TypesInfo, varTypes)
			case *ast.ExprStmt:
				recordResponseHeaderMutationsFromExpr(node.X, pkg.TypesInfo, &headerMutations)
			case *ast.ReturnStmt:
				commentDetection := true
				if resolver != nil {
					commentDetection = resolver.commentDetection
				}
				recordResponseTypeFromReturn(node, pkg, pkg.TypesInfo, &responses, resolver, commentDetection)
				recordResponseHeaderMutationsFromReturn(node, pkg.TypesInfo, &headerMutations)
			}
			return true
		})
	})

	if reqType != "" {
		out.Request = &RequestReport{
			Type:       reqType,
			Confidence: normalizeInferenceConfidence(reqConfidence),
		}
		if len(reqContentTypes) == 0 {
			reqContentTypes["application/json"] = struct{}{}
		}
		cts := make([]string, 0, len(reqContentTypes))
		for ct := range reqContentTypes {
			cts = append(cts, ct)
		}
		sort.Strings(cts)
		out.Request.ContentTypes = cts
	}
	if queryType != "" {
		out.Query = &TypeReport{
			Type:       queryType,
			Confidence: normalizeInferenceConfidence(queryConfidence),
		}
	}
	if len(pathTypes) > 0 {
		keys := make([]string, 0, len(pathTypes))
		for k := range pathTypes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out.Path = make([]PathParamReport, 0, len(keys))
		for _, k := range keys {
			p := pathTypes[k]
			p.Confidence = normalizeInferenceConfidence(p.Confidence)
			out.Path = append(out.Path, p)
		}
	}
	if len(responses) > 0 {
		responses = applyHeaderMutations(responses, headerMutations)
		out.Responses = toResponseTypeReports(responses)
	}
	return out
}

func joinRoutePath(prefix, p string) string {
	prefix = strings.TrimSpace(prefix)
	p = strings.TrimSpace(p)

	if prefix == "" {
		if p == "" {
			return "/"
		}
		if strings.HasPrefix(p, "/") {
			return p
		}
		return "/" + p
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if len(prefix) > 1 {
		prefix = strings.TrimSuffix(prefix, "/")
	}
	if p == "" || p == "/" {
		return prefix
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return prefix + p
}

func normalizeOpenAPIPathLocal(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		if strings.HasPrefix(segment, ":") && len(segment) > 1 {
			segments[i] = "{" + strings.TrimPrefix(segment, ":") + "}"
		}
	}
	return strings.Join(segments, "/")
}

func recordAssignTypes(node *ast.AssignStmt, info *types.Info, vars map[string]types.Type) {
	for i, lhs := range node.Lhs {
		ident, ok := lhs.(*ast.Ident)
		if !ok || ident.Name == "_" {
			continue
		}
		if i < len(node.Rhs) {
			if t := info.TypeOf(node.Rhs[i]); t != nil {
				vars[ident.Name] = t
				continue
			}
		}
		if obj := info.Defs[ident]; obj != nil {
			vars[ident.Name] = obj.Type()
		}
	}
}

func recordDeclTypes(node *ast.DeclStmt, info *types.Info, vars map[string]types.Type) {
	gen, ok := node.Decl.(*ast.GenDecl)
	if !ok || gen.Tok != token.VAR {
		return
	}
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for _, name := range vs.Names {
			if name.Name == "_" {
				continue
			}
			if obj := info.Defs[name]; obj != nil {
				vars[name.Name] = obj.Type()
			}
		}
	}
}

func recordPathParamAssignments(node *ast.AssignStmt, info *types.Info, paramVars map[string]string, pathTypes map[string]PathParamReport) {
	for i, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Params" {
			continue
		}
		name, ok := literalString(call.Args[0])
		if !ok {
			continue
		}
		if i >= len(node.Lhs) {
			continue
		}
		if ident, ok := node.Lhs[i].(*ast.Ident); ok {
			paramVars[ident.Name] = name
			if _, exists := pathTypes[name]; !exists {
				pathTypes[name] = PathParamReport{
					Name:       name,
					Type:       "string",
					Confidence: responseConfidenceInferred,
				}
			}
		}
	}

	for _, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || len(call.Args) == 0 {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			continue
		}
		pkgIdent, ok := sel.X.(*ast.Ident)
		if !ok || pkgIdent.Name != "strconv" {
			continue
		}
		arg, ok := call.Args[0].(*ast.Ident)
		if !ok {
			continue
		}
		paramName, ok := paramVars[arg.Name]
		if !ok {
			continue
		}
		paramMeta := pathTypes[paramName]
		paramMeta.Name = paramName
		paramMeta.Confidence = responseConfidenceExact
		switch sel.Sel.Name {
		case "ParseInt", "ParseUint":
			paramMeta.Type = "integer"
		case "ParseFloat":
			paramMeta.Type = "number"
		case "ParseBool":
			paramMeta.Type = "boolean"
		default:
			continue
		}
		pathTypes[paramName] = paramMeta
	}
}

func recordRequestType(node *ast.AssignStmt, info *types.Info, vars map[string]types.Type, out *string, confidence *string) {
	for _, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil || sel.Sel.Name != "Body" {
			continue
		}
		bindCall, ok := sel.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		bindSel, ok := bindCall.Fun.(*ast.SelectorExpr)
		if !ok || bindSel.Sel == nil || bindSel.Sel.Name != "Bind" {
			continue
		}
		arg := call.Args[0]
		if t, c := typeOfExprWithConfidence(arg, info, vars); t != nil {
			*out = canonicalTypeName(t)
			*confidence = c
		}
	}
}

func recordQueryType(node *ast.AssignStmt, info *types.Info, vars map[string]types.Type, out *string, confidence *string) {
	for _, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			continue
		}
		name := sel.Sel.Name
		if name != "Query" && name != "QueryParser" {
			continue
		}
		if t, c := typeOfExprWithConfidence(call.Args[0], info, vars); t != nil {
			*out = canonicalTypeName(t)
			*confidence = c
		}
	}
}

func recordResponseTypeFromAssign(node *ast.AssignStmt, info *types.Info, responses *[]detectedResponse, resolver *helperResolver) {
	for _, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		detected, ok := parseResponseCall(call, info, resolver)
		if !ok {
			continue
		}
		*responses = appendDetectedResponses(*responses, detected)
	}
}

func recordResponseTypeFromReturn(node *ast.ReturnStmt, pkg *packages.Package, info *types.Info, responses *[]detectedResponse, resolver *helperResolver, commentDetection bool) {
	description := extractNearbyCommentForNode(pkg, node, commentDetection)
	for _, expr := range node.Results {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			continue
		}
		detected, ok := parseResponseCall(call, info, resolver)
		if !ok {
			continue
		}
		detected = applyResponseDescription(detected, description)
		*responses = appendDetectedResponses(*responses, detected)
	}
}

func recordResponseHeaderMutationsFromAssign(node *ast.AssignStmt, info *types.Info, headers *[]detectedHeaderMutation) {
	for _, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		detected := parseHeaderMutationCall(call, info)
		*headers = appendHeaderMutations(*headers, detected)
	}
}

func recordResponseHeaderMutationsFromExpr(expr ast.Expr, info *types.Info, headers *[]detectedHeaderMutation) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return
	}
	detected := parseHeaderMutationCall(call, info)
	*headers = appendHeaderMutations(*headers, detected)
}

func recordResponseHeaderMutationsFromReturn(node *ast.ReturnStmt, info *types.Info, headers *[]detectedHeaderMutation) {
	for _, expr := range node.Results {
		call, ok := expr.(*ast.CallExpr)
		if !ok {
			continue
		}
		detected := parseHeaderMutationCall(call, info)
		*headers = appendHeaderMutations(*headers, detected)
	}
}

func parseHeaderMutationCall(call *ast.CallExpr, info *types.Info) []detectedHeaderMutation {
	baseFun := baseCallFunExpr(call.Fun)
	sel, ok := baseFun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return nil
	}

	switch sel.Sel.Name {
	case "Set":
		if len(call.Args) != 2 {
			return nil
		}
		name, ok := literalString(call.Args[0])
		if !ok {
			return nil
		}
		name = strings.TrimSpace(name)
		if name == "" || strings.EqualFold(name, "Content-Type") {
			return nil
		}
		status := 0
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, _ = extractStatusAndContentTypeFromChain(inner, info, status, "")
		}
		t := "string"
		if tv := info.TypeOf(call.Args[1]); tv != nil {
			t = openAPITypeFromTypeString(tv.String())
		}
		return []detectedHeaderMutation{{
			status: status,
			name:   name,
			header: ResponseHeaderReport{
				Type:       t,
				Confidence: responseConfidenceExact,
				Trace:      []string{"fiber.Ctx.Set"},
			},
		}}
	case "Cookie":
		status := 0
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, _ = extractStatusAndContentTypeFromChain(inner, info, status, "")
		}
		description := "Auto-detected"
		if len(call.Args) > 0 {
			if name := detectCookieName(call.Args[0]); name != "" {
				description = "Auto-detected cookie: " + name
			}
		}
		return []detectedHeaderMutation{{
			status: status,
			name:   "Set-Cookie",
			header: ResponseHeaderReport{
				Type:        "string",
				Description: description,
				Confidence:  responseConfidenceExact,
				Trace:       []string{"fiber.Ctx.Cookie"},
			},
		}}
	default:
		return nil
	}
}

func detectCookieName(expr ast.Expr) string {
	lit, ok := expr.(*ast.UnaryExpr)
	if ok && lit.Op == token.AND {
		expr = lit.X
	}
	cl, ok := expr.(*ast.CompositeLit)
	if !ok {
		return ""
	}
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok || keyIdent.Name != "Name" {
			continue
		}
		name, ok := literalString(kv.Value)
		if !ok {
			continue
		}
		return strings.TrimSpace(name)
	}
	return ""
}

func openAPITypeFromTypeString(typeName string) string {
	t := strings.ToLower(strings.TrimSpace(typeName))
	switch {
	case strings.HasPrefix(t, "int"), strings.HasPrefix(t, "uint"):
		return "integer"
	case strings.HasPrefix(t, "float"):
		return "number"
	case t == "bool":
		return "boolean"
	default:
		return "string"
	}
}

func appendHeaderMutations(existing []detectedHeaderMutation, in []detectedHeaderMutation) []detectedHeaderMutation {
	for _, item := range in {
		if strings.TrimSpace(item.name) == "" {
			continue
		}
		existing = append(existing, item)
	}
	return existing
}

func applyHeaderMutations(responses []detectedResponse, headers []detectedHeaderMutation) []detectedResponse {
	if len(responses) == 0 || len(headers) == 0 {
		return responses
	}
	out := make([]detectedResponse, 0, len(responses))
	for _, item := range responses {
		resp := item
		if resp.headers == nil {
			resp.headers = map[string]ResponseHeaderReport{}
		}
		for _, hm := range headers {
			if hm.status != 0 && hm.status != resp.status {
				continue
			}
			name := strings.TrimSpace(hm.name)
			if name == "" {
				continue
			}
			existing := resp.headers[name]
			if strings.TrimSpace(existing.Type) == "" {
				existing.Type = hm.header.Type
			}
			if strings.TrimSpace(existing.Description) == "" {
				existing.Description = hm.header.Description
			}
			existing.Confidence = normalizeInferenceConfidence(firstNonEmpty(existing.Confidence, hm.header.Confidence))
			if len(existing.Trace) == 0 && len(hm.header.Trace) > 0 {
				existing.Trace = append([]string(nil), hm.header.Trace...)
			}
			resp.headers[name] = existing
		}
		out = append(out, resp)
	}
	return out
}

func applyResponseDescription(in []detectedResponse, description string) []detectedResponse {
	description = strings.TrimSpace(description)
	if description == "" || len(in) == 0 {
		return in
	}
	out := make([]detectedResponse, 0, len(in))
	for _, item := range in {
		if strings.TrimSpace(item.description) == "" {
			item.description = description
		}
		out = append(out, item)
	}
	return out
}

func parseResponseCall(call *ast.CallExpr, info *types.Info, resolver *helperResolver) ([]detectedResponse, bool) {
	baseFun := baseCallFunExpr(call.Fun)
	sel, ok := baseFun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		if resolver != nil {
			if resolver.explicitOnly {
				if !isAllowedConcreteHelperResponseCall(call, info, resolver) {
					return nil, false
				}
			}
			if out, ok := resolver.resolveCall(call, info); ok {
				return downgradeResponsesConfidence(out, responseConfidenceInferred), true
			}
		}
		return nil, false
	}
	switch sel.Sel.Name {
	case "JSON":
		if len(call.Args) != 1 {
			return nil, false
		}
		status := 200
		contentType := "application/json"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		t := info.TypeOf(call.Args[0])
		if t == nil {
			return nil, false
		}
		return []detectedResponse{{status: status, typ: canonicalTypeName(t), contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.JSON"}, headers: headersToReportMap(headers)}}, true
	case "SendString":
		status := 200
		contentType := "text/plain"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		return []detectedResponse{{status: status, typ: "string", contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.SendString"}, headers: headersToReportMap(headers)}}, true
	case "Send":
		status := 200
		contentType := "application/octet-stream"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		return []detectedResponse{{status: status, typ: "string", contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.Send"}, headers: headersToReportMap(headers)}}, true
	case "SendStatus":
		status := 200
		if len(call.Args) == 1 {
			if code, ok := statusCodeFromExpr(call.Args[0], info); ok {
				status = code
			}
		}
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, _ = extractStatusAndContentTypeFromChain(inner, info, status, "")
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		return []detectedResponse{{status: status, typ: "", contentType: "", confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.SendStatus"}, headers: headersToReportMap(headers)}}, true
	case "SendFile", "SendStream", "SendStreamWriter", "Download":
		status := 200
		contentType := "application/octet-stream"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		return []detectedResponse{{status: status, typ: "string", contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx." + sel.Sel.Name}, headers: headersToReportMap(headers)}}, true
	case "XML":
		status := 200
		contentType := "application/xml"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		if len(call.Args) != 1 {
			return nil, false
		}
		t := info.TypeOf(call.Args[0])
		if t == nil {
			return nil, false
		}
		return []detectedResponse{{status: status, typ: canonicalTypeName(t), contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.XML"}, headers: headersToReportMap(headers)}}, true
	case "JSONP":
		status := 200
		contentType := "application/javascript"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		if len(call.Args) == 0 {
			return nil, false
		}
		t := info.TypeOf(call.Args[0])
		if t == nil {
			return nil, false
		}
		return []detectedResponse{{status: status, typ: canonicalTypeName(t), contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.JSONP"}, headers: headersToReportMap(headers)}}, true
	case "Render":
		status := 200
		contentType := "text/html"
		headers := []detectedHeaderMutation{}
		if inner, ok := sel.X.(*ast.CallExpr); ok {
			status, contentType = extractStatusAndContentTypeFromChain(inner, info, status, contentType)
			headers = extractHeaderMutationsFromChain(inner, info, status)
		}
		return []detectedResponse{{status: status, typ: "string", contentType: contentType, confidence: responseConfidenceExact, trace: []string{"fiber.Ctx.Render"}, headers: headersToReportMap(headers)}}, true
	default:
		if resolver != nil {
			if resolver.explicitOnly {
				if !isAllowedConcreteHelperResponseCall(call, info, resolver) {
					return nil, false
				}
			}
			if out, ok := resolver.resolveCall(call, info); ok {
				return downgradeResponsesConfidence(out, responseConfidenceInferred), true
			}
		}
		return nil, false
	}
}

func isAllowedConcreteHelperResponseCall(call *ast.CallExpr, info *types.Info, resolver *helperResolver) bool {
	if call == nil || info == nil {
		return false
	}
	if !callHasFiberCtxArg(call, info) {
		return false
	}
	baseFun := baseCallFunExpr(call.Fun)
	selExpr, ok := baseFun.(*ast.SelectorExpr)
	if !ok {
		// Allow direct helper function calls like sendJSON(c) in explicit-only mode.
		if _, isIdent := baseFun.(*ast.Ident); isIdent {
			return true
		}
		if resolver != nil {
			file, line, column, endLine, endColumn := resolver.lookupNodeSpan(info, call)
			resolver.addDiagnostic(
				"warning",
				"helper_response_dispatch_rejected",
				"response helper dispatch is limited to concrete method calls like *.Method(c, ...); use explicit c.* or annotate route output with .ProduceAs(...)",
				file, line, column, endLine, endColumn,
			)
		}
		return false
	}
	sel := info.Selections[selExpr]
	if sel == nil || sel.Obj() == nil {
		if resolver != nil {
			file, line, column, endLine, endColumn := resolver.lookupNodeSpan(info, selExpr)
			resolver.addDiagnostic(
				"warning",
				"helper_response_dispatch_rejected",
				"response helper dispatch is limited to concrete method calls like *.Method(c, ...); use explicit c.* or annotate route output with .ProduceAs(...)",
				file, line, column, endLine, endColumn,
			)
		}
		return false
	}
	recv := sel.Recv()
	if recv == nil {
		return false
	}
	if _, iface := recv.Underlying().(*types.Interface); iface {
		if resolver != nil {
			file, line, column, endLine, endColumn := resolver.lookupNodeSpan(info, selExpr)
			resolver.addDiagnostic(
				"warning",
				"helper_response_dispatch_ambiguous",
				fmt.Sprintf("skipping non-concrete helper dispatch for %s; use .ProduceAs(...) on the route to declare response type", recv.String()),
				file, line, column, endLine, endColumn,
			)
		}
		return false
	}
	return true
}

func callHasFiberCtxArg(call *ast.CallExpr, info *types.Info) bool {
	if call == nil || info == nil {
		return false
	}
	for _, arg := range call.Args {
		t := info.TypeOf(arg)
		if t == nil {
			continue
		}
		if t.String() == "github.com/gofiber/fiber/v3.Ctx" {
			return true
		}
	}
	return false
}

func baseCallFunExpr(expr ast.Expr) ast.Expr {
	for expr != nil {
		switch v := expr.(type) {
		case *ast.IndexExpr:
			expr = v.X
		case *ast.IndexListExpr:
			expr = v.X
		case *ast.ParenExpr:
			expr = v.X
		default:
			return expr
		}
	}
	return nil
}

func extractStatusAndContentTypeFromChain(call *ast.CallExpr, info *types.Info, status int, contentType string) (int, string) {
	cur := call
	for cur != nil {
		sel, ok := cur.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			break
		}
		switch sel.Sel.Name {
		case "Status":
			if len(cur.Args) == 1 {
				if code, ok := statusCodeFromExpr(cur.Args[0], info); ok {
					status = code
				}
			}
		case "Set":
			if len(cur.Args) == 2 {
				if k, ok := literalString(cur.Args[0]); ok && strings.EqualFold(strings.TrimSpace(k), "Content-Type") {
					if v, ok := literalString(cur.Args[1]); ok {
						contentType = strings.TrimSpace(v)
					}
				}
			}
		case "Type":
			if len(cur.Args) > 0 {
				if ext, ok := literalString(cur.Args[0]); ok {
					if guessed := contentTypeFromExt(ext); guessed != "" {
						contentType = guessed
					}
				}
			}
		}
		next, ok := sel.X.(*ast.CallExpr)
		if !ok {
			break
		}
		cur = next
	}
	return status, contentType
}

func extractHeaderMutationsFromChain(call *ast.CallExpr, info *types.Info, status int) []detectedHeaderMutation {
	out := []detectedHeaderMutation{}
	cur := call
	for cur != nil {
		sel, ok := cur.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			break
		}
		switch sel.Sel.Name {
		case "Set":
			if len(cur.Args) == 2 {
				name, ok := literalString(cur.Args[0])
				if ok {
					name = strings.TrimSpace(name)
					if name != "" && !strings.EqualFold(name, "Content-Type") {
						t := "string"
						if tv := info.TypeOf(cur.Args[1]); tv != nil {
							t = openAPITypeFromTypeString(tv.String())
						}
						out = append(out, detectedHeaderMutation{
							status: status,
							name:   name,
							header: ResponseHeaderReport{
								Type:       t,
								Confidence: responseConfidenceExact,
								Trace:      []string{"fiber.Ctx.Set"},
							},
						})
					}
				}
			}
		case "Cookie":
			description := "Auto-detected"
			if len(cur.Args) > 0 {
				if name := detectCookieName(cur.Args[0]); name != "" {
					description = "Auto-detected cookie: " + name
				}
			}
			out = append(out, detectedHeaderMutation{
				status: status,
				name:   "Set-Cookie",
				header: ResponseHeaderReport{
					Type:        "string",
					Description: description,
					Confidence:  responseConfidenceExact,
					Trace:       []string{"fiber.Ctx.Cookie"},
				},
			})
		}
		next, ok := sel.X.(*ast.CallExpr)
		if !ok {
			break
		}
		cur = next
	}
	return out
}

func headersToReportMap(items []detectedHeaderMutation) map[string]ResponseHeaderReport {
	if len(items) == 0 {
		return nil
	}
	out := map[string]ResponseHeaderReport{}
	for _, item := range items {
		name := strings.TrimSpace(item.name)
		if name == "" {
			continue
		}
		existing := out[name]
		if strings.TrimSpace(existing.Type) == "" {
			existing.Type = item.header.Type
		}
		if strings.TrimSpace(existing.Description) == "" {
			existing.Description = item.header.Description
		}
		existing.Confidence = normalizeInferenceConfidence(firstNonEmpty(existing.Confidence, item.header.Confidence))
		if len(existing.Trace) == 0 && len(item.header.Trace) > 0 {
			existing.Trace = append([]string(nil), item.header.Trace...)
		}
		out[name] = existing
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func contentTypeFromExt(ext string) string {
	switch strings.ToLower(strings.TrimSpace(ext)) {
	case "json":
		return "application/json"
	case "xml":
		return "application/xml"
	case "html":
		return "text/html"
	case "txt", "text":
		return "text/plain"
	case "csv":
		return "text/csv"
	default:
		return ""
	}
}

func recordRequestContentType(node *ast.AssignStmt, contentTypes map[string]struct{}) {
	for _, rhs := range node.Rhs {
		call, ok := rhs.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel == nil {
			continue
		}
		switch sel.Sel.Name {
		case "Body":
			if bindCall, ok := sel.X.(*ast.CallExpr); ok {
				if bindSel, ok := bindCall.Fun.(*ast.SelectorExpr); ok && bindSel.Sel != nil && bindSel.Sel.Name == "Bind" {
					contentTypes["application/json"] = struct{}{}
				}
			}
		case "Form":
			if bindCall, ok := sel.X.(*ast.CallExpr); ok {
				if bindSel, ok := bindCall.Fun.(*ast.SelectorExpr); ok && bindSel.Sel != nil && bindSel.Sel.Name == "Bind" {
					contentTypes["application/x-www-form-urlencoded"] = struct{}{}
				}
			}
		}
		if sel.Sel.Name == "FormValue" {
			contentTypes["application/x-www-form-urlencoded"] = struct{}{}
		}
		if sel.Sel.Name == "FormFile" || sel.Sel.Name == "MultipartForm" {
			contentTypes["multipart/form-data"] = struct{}{}
		}
	}
}

func collectAllPackages(roots []*packages.Package) []*packages.Package {
	seen := map[string]bool{}
	out := make([]*packages.Package, 0, len(roots))
	var walk func(p *packages.Package)
	walk = func(p *packages.Package) {
		if p == nil || seen[p.ID] {
			return
		}
		seen[p.ID] = true
		out = append(out, p)
		for _, imp := range p.Imports {
			walk(imp)
		}
	}
	for _, p := range roots {
		walk(p)
	}
	return out
}

func buildGlobalHelperDeclIndex(pkgs []*packages.Package) map[string]helperFuncDecl {
	index := map[string]helperFuncDecl{}
	for _, pkg := range pkgs {
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Body == nil {
					continue
				}
				if obj := pkg.TypesInfo.Defs[fn.Name]; obj != nil {
					if f, ok := obj.(*types.Func); ok {
						index[f.FullName()] = helperFuncDecl{
							pkg: pkg,
							fn:  fn,
						}
					}
				}
			}
		}
	}
	return index
}

func buildReferencedHelperDeclIndex(scopedPkgs, roots []*packages.Package, all map[string]helperFuncDecl, provided []providedBinding) map[string]helperFuncDecl {
	if len(all) == 0 {
		return all
	}
	seeds := collectEntrypointSeeds(roots)
	if len(seeds) == 0 {
		seeds = collectHandlerSeeds(roots)
	}

	for _, call := range collectCallsReachableFromUsNew(roots, all) {
		if key, ok := resolveCallExprFuncKey(call.call, call.info); ok {
			seeds[key] = struct{}{}
		}
	}

	for _, b := range provided {
		for _, key := range lookupProviderMethodKeysFromBinding(b, all) {
			seeds[key] = struct{}{}
		}
	}

	queue := make([]string, 0, len(seeds))
	for key := range seeds {
		if _, ok := all[key]; ok {
			queue = append(queue, key)
		}
	}

	referenced := map[string]helperFuncDecl{}
	seen := map[string]struct{}{}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		decl, ok := all[key]
		if !ok || decl.fn == nil || decl.pkg == nil || decl.pkg.TypesInfo == nil || decl.fn.Body == nil {
			continue
		}
		referenced[key] = decl
		ast.Inspect(decl.fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if next, ok := resolveCallExprFuncKey(call, decl.pkg.TypesInfo); ok {
				if _, exists := seen[next]; !exists {
					if _, known := all[next]; known {
						queue = append(queue, next)
					}
				}
			}
			if shouldCollectFunctionArgsFromCall(call, decl.pkg.TypesInfo) {
				for _, arg := range call.Args {
					if next, ok := resolveFuncExprKey(arg, decl.pkg.TypesInfo); ok {
						if _, exists := seen[next]; !exists {
							if _, known := all[next]; known {
								queue = append(queue, next)
							}
						}
					}
				}
			}
			return true
		})
	}
	return referenced
}

func collectEntrypointSeeds(roots []*packages.Package) map[string]struct{} {
	seeds := map[string]struct{}{}
	for _, pkg := range roots {
		if pkg == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			if file == nil {
				continue
			}
			filename := filepath.Base(pkg.Fset.Position(file.Pos()).Filename)
			isEntrypointFile := strings.EqualFold(strings.TrimSpace(filename), "main.go")
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Body == nil {
					continue
				}
				name := strings.TrimSpace(fn.Name.Name)
				if !isEntrypointFile && !(name == "main" && strings.TrimSpace(pkg.Name) == "main") {
					continue
				}
				if obj := pkg.TypesInfo.Defs[fn.Name]; obj != nil {
					if f, ok := obj.(*types.Func); ok {
						seeds[f.FullName()] = struct{}{}
					}
				}
			}
		}
	}
	return seeds
}

func collectHandlerSeeds(roots []*packages.Package) map[string]struct{} {
	seeds := map[string]struct{}{}
	for _, pkg := range roots {
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name == nil || fn.Body == nil {
					continue
				}
				if isFiberCtxHandler(fn, pkg.TypesInfo) || isRouterHandleMethod(fn, pkg.TypesInfo) {
					if obj := pkg.TypesInfo.Defs[fn.Name]; obj != nil {
						if f, ok := obj.(*types.Func); ok {
							seeds[f.FullName()] = struct{}{}
						}
					}
				}
			}
		}
	}
	return seeds
}

func buildDiagnosticSourceIndex(pkgs []*packages.Package) map[*types.Info]diagnosticSource {
	out := map[*types.Info]diagnosticSource{}
	for _, pkg := range pkgs {
		if pkg == nil || pkg.TypesInfo == nil || pkg.Fset == nil {
			continue
		}
		out[pkg.TypesInfo] = diagnosticSource{
			pkgPath: pkg.PkgPath,
			fset:    pkg.Fset,
		}
	}
	return out
}

func buildDIProvidedBindings(allPkgs []*packages.Package, roots []*packages.Package, globalDecls map[string]helperFuncDecl) []providedBinding {
	scopedCalls := collectCallsReachableFromUsNew(roots, globalDecls)
	if len(scopedCalls) == 0 {
		return buildDIProvidedBindingsUnscoped(allPkgs)
	}
	out := make([]providedBinding, 0, len(scopedCalls))
	for _, call := range scopedCalls {
		if binding, ok := providedBindingFromProvideCall(call.call, call.info); ok {
			out = append(out, binding)
		}
	}
	return dedupeProvidedBindings(out)
}

type typedCallExpr struct {
	call *ast.CallExpr
	info *types.Info
}

func buildDIProvidedBindingsUnscoped(pkgs []*packages.Package) []providedBinding {
	out := make([]providedBinding, 0, 32)
	for _, pkg := range pkgs {
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || len(call.Args) == 0 {
					return true
				}
				if !isDIProvideCall(call, pkg.TypesInfo) {
					return true
				}
				if binding, ok := providedBindingFromProvideCall(call, pkg.TypesInfo); ok {
					out = append(out, binding)
				}
				return true
			})
		}
	}
	return dedupeProvidedBindings(out)
}

func collectCallsReachableFromUsNew(roots []*packages.Package, globalDecls map[string]helperFuncDecl) []typedCallExpr {
	if len(roots) == 0 || len(globalDecls) == 0 {
		return nil
	}
	seedKeys := map[string]struct{}{}
	for _, pkg := range roots {
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		for _, file := range pkg.Syntax {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || fn.Name == nil {
					continue
				}
				hasUSNew := false
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					if isUSNewCall(call, pkg.TypesInfo) {
						hasUSNew = true
					}
					return true
				})
				if !hasUSNew {
					continue
				}
				if obj := pkg.TypesInfo.Defs[fn.Name]; obj != nil {
					if f, ok := obj.(*types.Func); ok {
						seedKeys[f.FullName()] = struct{}{}
					}
				}
			}
		}
	}
	if len(seedKeys) == 0 {
		return nil
	}

	out := []typedCallExpr{}
	seenFunc := map[string]struct{}{}
	queue := make([]string, 0, len(seedKeys))
	for key := range seedKeys {
		queue = append(queue, key)
	}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, ok := seenFunc[key]; ok {
			continue
		}
		seenFunc[key] = struct{}{}
		decl, ok := globalDecls[key]
		if !ok || decl.fn == nil || decl.pkg == nil || decl.pkg.TypesInfo == nil || decl.fn.Body == nil {
			continue
		}
		ast.Inspect(decl.fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if isDIProvideCall(call, decl.pkg.TypesInfo) {
				out = append(out, typedCallExpr{
					call: call,
					info: decl.pkg.TypesInfo,
				})
			}
			if calledKey, ok := resolveCallExprFuncKey(call, decl.pkg.TypesInfo); ok {
				if _, exists := seenFunc[calledKey]; !exists {
					queue = append(queue, calledKey)
				}
			}
			return true
		})
	}
	return out
}

func isUSNewCall(call *ast.CallExpr, info *types.Info) bool {
	if call == nil || info == nil {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "New" {
		return false
	}
	if ident, ok := sel.X.(*ast.Ident); ok {
		if used := info.Uses[ident]; used != nil {
			if pkgName, ok := used.(*types.PkgName); ok && pkgName.Imported() != nil {
				return pkgName.Imported().Path() == "github.com/bronystylecrazy/ultrastructure"
			}
		}
	}
	return false
}

func resolveCallExprFuncKey(call *ast.CallExpr, info *types.Info) (string, bool) {
	if call == nil || info == nil {
		return "", false
	}
	switch f := baseCallFunExpr(call.Fun).(type) {
	case *ast.Ident:
		if obj := info.Uses[f]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				return fn.FullName(), true
			}
		}
	case *ast.SelectorExpr:
		if sel := info.Selections[f]; sel != nil && sel.Obj() != nil {
			if fn, ok := sel.Obj().(*types.Func); ok {
				return fn.FullName(), true
			}
		}
		if f.Sel != nil {
			if obj := info.Uses[f.Sel]; obj != nil {
				if fn, ok := obj.(*types.Func); ok {
					return fn.FullName(), true
				}
			}
		}
	}
	return "", false
}

func resolveFuncExprKey(expr ast.Expr, info *types.Info) (string, bool) {
	if expr == nil || info == nil {
		return "", false
	}
	switch v := expr.(type) {
	case *ast.Ident:
		if obj := info.Uses[v]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				key := strings.TrimSpace(fn.FullName())
				return key, key != ""
			}
		}
	case *ast.SelectorExpr:
		if sel := info.Selections[v]; sel != nil && sel.Obj() != nil {
			if fn, ok := sel.Obj().(*types.Func); ok {
				key := strings.TrimSpace(fn.FullName())
				return key, key != ""
			}
		}
		if obj := info.Uses[v.Sel]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				key := strings.TrimSpace(fn.FullName())
				return key, key != ""
			}
		}
	case *ast.IndexExpr:
		return resolveFuncExprKey(v.X, info)
	case *ast.IndexListExpr:
		return resolveFuncExprKey(v.X, info)
	case *ast.ParenExpr:
		return resolveFuncExprKey(v.X, info)
	}
	return "", false
}

func shouldCollectFunctionArgsFromCall(call *ast.CallExpr, info *types.Info) bool {
	if call == nil || info == nil {
		return false
	}
	if key, ok := resolveCallExprFuncKey(call, info); ok {
		switch key {
		case "github.com/bronystylecrazy/ultrastructure.New",
			"github.com/bronystylecrazy/ultrastructure/di.Provide":
			return true
		}
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return false
	}
	switch strings.ToUpper(strings.TrimSpace(sel.Sel.Name)) {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS", "ALL":
		return true
	case "USE", "GROUP":
		return true
	}
	return false
}

func isDIProvideCall(call *ast.CallExpr, info *types.Info) bool {
	if call == nil || info == nil {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "Provide" {
		return false
	}
	if ident, ok := sel.X.(*ast.Ident); ok {
		if used := info.Uses[ident]; used != nil {
			if pkgName, ok := used.(*types.PkgName); ok && pkgName.Imported() != nil {
				return pkgName.Imported().Path() == "github.com/bronystylecrazy/ultrastructure/di"
			}
		}
	}
	return false
}

func providedTypesFromConstructorExpr(expr ast.Expr, info *types.Info) []types.Type {
	if expr == nil || info == nil {
		return nil
	}
	typ := info.TypeOf(expr)
	sig, ok := typ.(*types.Signature)
	if !ok || sig.Results() == nil {
		return nil
	}
	out := make([]types.Type, 0, sig.Results().Len())
	for i := 0; i < sig.Results().Len(); i++ {
		t := sig.Results().At(i).Type()
		if isErrorType(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func providedBindingFromProvideCall(call *ast.CallExpr, info *types.Info) (providedBinding, bool) {
	if call == nil || info == nil || len(call.Args) == 0 {
		return providedBinding{}, false
	}
	base := providedTypesFromConstructorExpr(call.Args[0], info)
	if len(base) == 0 {
		return providedBinding{}, false
	}
	binding := providedBinding{
		Concrete: base[0],
	}
	for _, opt := range call.Args[1:] {
		export, includeSelf, ok := parseDIProvideOption(opt, info)
		if includeSelf {
			binding.IncludeSelf = true
		}
		if ok && export != nil {
			binding.Exports = append(binding.Exports, export)
		}
	}
	if len(binding.Exports) == 0 {
		binding.IncludeSelf = true
	}
	binding.Exports = dedupeProvidedTypes(binding.Exports)
	return binding, true
}

func parseDIProvideOption(expr ast.Expr, info *types.Info) (types.Type, bool, bool) {
	if expr == nil || info == nil {
		return nil, false, false
	}
	switch v := expr.(type) {
	case *ast.CallExpr:
		switch fn := v.Fun.(type) {
		case *ast.SelectorExpr:
			if fn.Sel == nil {
				return nil, false, false
			}
			if pkgIdent, ok := fn.X.(*ast.Ident); ok {
				if used := info.Uses[pkgIdent]; used != nil {
					if pkgName, ok := used.(*types.PkgName); ok && pkgName.Imported() != nil && pkgName.Imported().Path() == "github.com/bronystylecrazy/ultrastructure/di" {
						switch fn.Sel.Name {
						case "Self":
							return nil, true, false
						}
					}
				}
			}
		case *ast.IndexExpr:
			if sel, ok := fn.X.(*ast.SelectorExpr); ok && sel.Sel != nil {
				if pkgIdent, ok := sel.X.(*ast.Ident); ok {
					if used := info.Uses[pkgIdent]; used != nil {
						if pkgName, ok := used.(*types.PkgName); ok && pkgName.Imported() != nil && pkgName.Imported().Path() == "github.com/bronystylecrazy/ultrastructure/di" {
							switch sel.Sel.Name {
							case "As":
								return info.TypeOf(fn.Index), false, true
							case "AsSelf":
								return info.TypeOf(fn.Index), true, true
							}
						}
					}
				}
			}
		}
	}
	return nil, false, false
}

func isErrorType(t types.Type) bool {
	if t == nil {
		return false
	}
	if named, ok := t.(*types.Named); ok && named.Obj() != nil && named.Obj().Pkg() == nil && named.Obj().Name() == "error" {
		return true
	}
	return t.String() == "error"
}

func dedupeProvidedTypes(in []types.Type) []types.Type {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]types.Type, 0, len(in))
	for _, t := range in {
		if t == nil {
			continue
		}
		key := t.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

func dedupeProvidedBindings(in []providedBinding) []providedBinding {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]providedBinding, 0, len(in))
	for _, b := range in {
		if b.Concrete == nil {
			continue
		}
		expKeys := make([]string, 0, len(b.Exports))
		for _, t := range b.Exports {
			if t != nil {
				expKeys = append(expKeys, t.String())
			}
		}
		sort.Strings(expKeys)
		key := b.Concrete.String() + "|self=" + strconv.FormatBool(b.IncludeSelf) + "|exp=" + strings.Join(expKeys, ",")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, b)
	}
	return out
}

func buildDependencyGraph(bindings []providedBinding) *DependencyGraph {
	if len(bindings) == 0 {
		return nil
	}
	nodeByID := map[string]DependencyGraphNode{}
	edges := make([]DependencyGraphEdge, 0, len(bindings)*2)
	edgeSeen := map[string]struct{}{}

	addNode := func(id, kind, label string) {
		if strings.TrimSpace(id) == "" {
			return
		}
		if _, exists := nodeByID[id]; exists {
			return
		}
		nodeByID[id] = DependencyGraphNode{
			ID:    id,
			Kind:  kind,
			Label: label,
		}
	}
	addEdge := func(from, to, kind string) {
		if from == "" || to == "" {
			return
		}
		key := from + "|" + to + "|" + kind
		if _, exists := edgeSeen[key]; exists {
			return
		}
		edgeSeen[key] = struct{}{}
		edges = append(edges, DependencyGraphEdge{
			From: from,
			To:   to,
			Kind: kind,
		})
	}

	for _, b := range bindings {
		if b.Concrete == nil {
			continue
		}
		concreteID := "type:" + b.Concrete.String()
		addNode(concreteID, "concrete", b.Concrete.String())

		if b.IncludeSelf {
			addEdge(concreteID, concreteID, "provides_self")
		}
		for _, exp := range b.Exports {
			if exp == nil {
				continue
			}
			exportID := "type:" + exp.String()
			kind := "export"
			if _, ok := exp.Underlying().(*types.Interface); ok {
				kind = "interface"
			}
			addNode(exportID, kind, exp.String())
			addEdge(concreteID, exportID, "provides_as")
		}
	}

	nodes := make([]DependencyGraphNode, 0, len(nodeByID))
	for _, n := range nodeByID {
		nodes = append(nodes, n)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].Kind < edges[j].Kind
	})

	return &DependencyGraph{
		Nodes: nodes,
		Edges: edges,
	}
}

func candidateVariants(t types.Type) []types.Type {
	if t == nil {
		return nil
	}
	out := []types.Type{t}
	switch tt := t.(type) {
	case *types.Pointer:
		out = append(out, tt.Elem())
	default:
		out = append(out, types.NewPointer(t))
	}
	dedup := make([]types.Type, 0, len(out))
	seen := map[string]struct{}{}
	for _, item := range out {
		if item == nil {
			continue
		}
		key := item.String()
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dedup = append(dedup, item)
	}
	return dedup
}

func newHelperResolver(globalDecls map[string]helperFuncDecl, provided []providedBinding, pkgPath string, strictDI bool, explicitOnly bool, commentDetection bool, sourceByInfo map[*types.Info]diagnosticSource) *helperResolver {
	if len(globalDecls) == 0 {
		return nil
	}
	return &helperResolver{
		declByKey:     globalDecls,
		provided:      dedupeProvidedBindings(provided),
		pkgPath:       pkgPath,
		strictDI:      strictDI,
		explicitOnly:  explicitOnly,
		diagnostics:   []AnalyzerDiagnostic{},
		diagSeen:      map[string]struct{}{},
		sourceByInfo:  sourceByInfo,
		inStack:       map[string]bool{},
		cache:         map[string][]detectedResponse{},
		dispatchCache: map[string][]string{},
		commentDetection: commentDetection,
	}
}

func (r *helperResolver) resolveCall(call *ast.CallExpr, info *types.Info) ([]detectedResponse, bool) {
	if r == nil || call == nil || info == nil {
		return nil, false
	}
	key := ""
	switch f := baseCallFunExpr(call.Fun).(type) {
	case *ast.Ident:
		if obj := info.Uses[f]; obj != nil {
			if fn, ok := obj.(*types.Func); ok {
				key = fn.FullName()
			}
		}
	case *ast.SelectorExpr:
		if sel := info.Selections[f]; sel != nil && sel.Obj() != nil {
			if fn, ok := sel.Obj().(*types.Func); ok {
				key = fn.FullName()
			}
			if out, ok := r.resolveSelectorDispatch(f, sel, info); ok {
				return out, true
			}
		}
		if key == "" && f.Sel != nil {
			if obj := info.Uses[f.Sel]; obj != nil {
				if fn, ok := obj.(*types.Func); ok {
					key = fn.FullName()
				}
			}
		}
	}
	if key == "" {
		return nil, false
	}
	out, ok := r.resolveFuncByKey(key)
	if !ok {
		return nil, false
	}
	out = applyGenericCallTypeArgs(call, info, out)
	return prependTrace(out, "call:"+key), true
}

func applyGenericCallTypeArgs(call *ast.CallExpr, info *types.Info, in []detectedResponse) []detectedResponse {
	if len(in) == 0 || call == nil || info == nil {
		return in
	}
	paramToConcrete := map[string]string{}
	var baseFun ast.Expr
	var typeArgExprs []ast.Expr
	switch f := call.Fun.(type) {
	case *ast.IndexExpr:
		baseFun = f.X
		typeArgExprs = []ast.Expr{f.Index}
	case *ast.IndexListExpr:
		baseFun = f.X
		typeArgExprs = append([]ast.Expr(nil), f.Indices...)
	default:
		return in
	}
	if len(typeArgExprs) == 0 {
		return in
	}
	var fn *types.Func
	switch v := baseCallFunExpr(baseFun).(type) {
	case *ast.Ident:
		if obj := info.Uses[v]; obj != nil {
			if f, ok := obj.(*types.Func); ok {
				fn = f
			}
		}
	case *ast.SelectorExpr:
		if sel := info.Selections[v]; sel != nil && sel.Obj() != nil {
			if f, ok := sel.Obj().(*types.Func); ok {
				fn = f
			}
		}
		if fn == nil && v.Sel != nil {
			if obj := info.Uses[v.Sel]; obj != nil {
				if f, ok := obj.(*types.Func); ok {
					fn = f
				}
			}
		}
	}
	if fn == nil {
		return in
	}
	sig, ok := fn.Type().(*types.Signature)
	if !ok || sig.TypeParams() == nil || sig.TypeParams().Len() == 0 {
		return in
	}
	n := sig.TypeParams().Len()
	if len(typeArgExprs) < n {
		n = len(typeArgExprs)
	}
	for i := 0; i < n; i++ {
		param := sig.TypeParams().At(i)
		if param == nil || param.Obj() == nil {
			continue
		}
		t := info.TypeOf(typeArgExprs[i])
		if t == nil {
			continue
		}
		paramToConcrete[param.Obj().Name()] = canonicalTypeName(t)
	}
	if len(paramToConcrete) == 0 {
		return in
	}
	out := make([]detectedResponse, 0, len(in))
	for _, item := range in {
		if repl, ok := paramToConcrete[item.typ]; ok {
			item.typ = repl
		} else if strings.HasPrefix(item.typ, "*") {
			if repl, ok := paramToConcrete[strings.TrimPrefix(item.typ, "*")]; ok {
				item.typ = "*" + repl
			}
		}
		out = append(out, item)
	}
	return out
}

func (r *helperResolver) resolveSelectorDispatch(selExpr *ast.SelectorExpr, sel *types.Selection, info *types.Info) ([]detectedResponse, bool) {
	if r == nil || selExpr == nil || sel == nil || info == nil || selExpr.Sel == nil {
		return nil, false
	}
	recv := sel.Recv()
	if recv == nil {
		return nil, false
	}
	if _, ok := recv.Underlying().(*types.Interface); !ok {
		return nil, false
	}
	methodName := strings.TrimSpace(selExpr.Sel.Name)
	if methodName == "" {
		return nil, false
	}

	dispatchKey := recv.String() + "::" + methodName
	if keys, ok := r.dispatchCache[dispatchKey]; ok {
		return r.resolveResponsesForDispatchKeys(keys)
	}

	keys := r.lookupProviderMethodKeys(recv, methodName)
	r.dispatchCache[dispatchKey] = keys
	file, line, column, endLine, endColumn := r.lookupNodeSpan(info, selExpr)
	if len(keys) == 0 {
		severity := "warning"
		if r.strictDI {
			severity = "error"
		}
		r.addDiagnostic(severity, "di_unresolved_interface_dispatch", fmt.Sprintf("unable to resolve interface dispatch for %s", dispatchKey), file, line, column, endLine, endColumn)
	} else if len(keys) > 1 {
		severity := "warning"
		if r.strictDI {
			severity = "error"
		}
		r.addDiagnostic(severity, "di_ambiguous_interface_dispatch", fmt.Sprintf("multiple DI candidates for %s (%d candidates)", dispatchKey, len(keys)), file, line, column, endLine, endColumn)
	}
	return r.resolveResponsesForDispatchKeys(keys)
}

func (r *helperResolver) lookupProviderMethodKeys(iface types.Type, methodName string) []string {
	if r == nil {
		return nil
	}
	name := strings.TrimSpace(methodName)
	if name == "" {
		return nil
	}
	keys := make([]string, 0, 2)
	seen := map[string]struct{}{}
	for _, binding := range r.provided {
		if !bindingMatchesInterface(binding, iface) {
			continue
		}
		for _, key := range lookupProviderMethodKeysFromBindingAndMethod(binding, iface, name) {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func lookupProviderMethodKeysFromBinding(binding providedBinding, decls map[string]helperFuncDecl) []string {
	keys := map[string]struct{}{}
	for _, exp := range binding.Exports {
		if exp == nil {
			continue
		}
		iface, ok := exp.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		for i := 0; i < iface.NumMethods(); i++ {
			for _, key := range lookupProviderMethodKeysFromBindingAndMethod(binding, exp, iface.Method(i).Name()) {
				if _, exists := decls[key]; exists {
					keys[key] = struct{}{}
				}
			}
		}
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func lookupProviderMethodKeysFromBindingAndMethod(binding providedBinding, iface types.Type, methodName string) []string {
	if binding.Concrete == nil || iface == nil {
		return nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, variant := range candidateVariants(binding.Concrete) {
		if !types.AssignableTo(variant, iface) {
			continue
		}
		obj, _, _ := types.LookupFieldOrMethod(variant, true, nil, methodName)
		fn, ok := obj.(*types.Func)
		if !ok || fn == nil {
			continue
		}
		key := fn.FullName()
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func bindingMatchesInterface(binding providedBinding, iface types.Type) bool {
	if binding.Concrete == nil || iface == nil {
		return false
	}
	if binding.IncludeSelf {
		for _, variant := range candidateVariants(binding.Concrete) {
			if types.AssignableTo(variant, iface) {
				return true
			}
		}
	}
	for _, exp := range binding.Exports {
		if exp == nil {
			continue
		}
		if types.Identical(exp, iface) {
			return true
		}
		if types.AssignableTo(exp, iface) {
			return true
		}
	}
	return false
}

func (r *helperResolver) addDiagnostic(severity, code, message, file string, line, column, endLine, endColumn int) {
	if r == nil {
		return
	}
	severity = strings.TrimSpace(strings.ToLower(severity))
	if severity != "error" {
		severity = "warning"
	}
	code = strings.TrimSpace(code)
	message = strings.TrimSpace(message)
	if code == "" || message == "" {
		return
	}
	key := severity + "|" + code + "|" + message + "|" + file + "|" + strconv.Itoa(line) + "|" + strconv.Itoa(column)
	if _, exists := r.diagSeen[key]; exists {
		return
	}
	r.diagSeen[key] = struct{}{}
	lineText, caret := readLineAndCaret(file, line, column, endLine, endColumn)
	r.diagnostics = append(r.diagnostics, AnalyzerDiagnostic{
		Severity:   severity,
		Code:       code,
		Message:    message,
		Package:    r.pkgPath,
		HandlerKey: strings.TrimSpace(r.handlerKey),
		File:       file,
		Line:       line,
		Column:     column,
		LineText:   lineText,
		Caret:      caret,
	})
}

func (r *helperResolver) lookupNodeSpan(info *types.Info, node ast.Node) (string, int, int, int, int) {
	if r == nil || info == nil || node == nil {
		return "", 0, 0, 0, 0
	}
	src, ok := r.sourceByInfo[info]
	if !ok || src.fset == nil {
		return "", 0, 0, 0, 0
	}
	pos := src.fset.Position(node.Pos())
	end := src.fset.Position(node.End())
	if !pos.IsValid() {
		return "", 0, 0, 0, 0
	}
	return pos.Filename, pos.Line, pos.Column, end.Line, end.Column
}

func readLineAndCaret(file string, line, column, endLine, endColumn int) (string, string) {
	file = strings.TrimSpace(file)
	if file == "" || line <= 0 || column <= 0 {
		return "", ""
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", ""
	}
	lines := strings.Split(string(b), "\n")
	if line > len(lines) {
		return "", ""
	}
	lineText := lines[line-1]
	if lineText == "" {
		return "", ""
	}
	start := clampColumn(column, lineText)
	end := start + 1
	if endLine == line && endColumn > start {
		end = clampColumn(endColumn, lineText)
	}
	if end <= start {
		end = start + 1
	}
	visualStart := toVisualColumn(lineText, start)
	visualEnd := toVisualColumn(lineText, end)
	if visualEnd <= visualStart {
		visualEnd = visualStart + 1
	}
	prefixSpaces := strings.Repeat(" ", visualStart-1)
	caret := prefixSpaces + strings.Repeat("^", visualEnd-visualStart)
	return lineText, caret
}

func clampColumn(column int, lineText string) int {
	if column < 1 {
		return 1
	}
	max := len([]rune(lineText)) + 1
	if column > max {
		return max
	}
	return column
}

func toVisualColumn(lineText string, runeColumn int) int {
	if runeColumn <= 1 {
		return 1
	}
	const tabWidth = 8
	visual := 1
	runes := []rune(lineText)
	limit := runeColumn - 1
	if limit > len(runes) {
		limit = len(runes)
	}
	for i := 0; i < limit; i++ {
		if runes[i] == '\t' {
			next := ((visual-1)/tabWidth + 1) * tabWidth
			visual = next + 1
			continue
		}
		visual++
	}
	return visual
}

func (r *helperResolver) resolveResponsesForDispatchKeys(keys []string) ([]detectedResponse, bool) {
	if len(keys) == 0 {
		return nil, false
	}
	merged := []detectedResponse{}
	for _, key := range keys {
		out, ok := r.resolveFuncByKey(key)
		if !ok {
			continue
		}
		merged = appendDetectedResponses(merged, prependTrace(out, "candidate:"+key))
	}
	if len(merged) == 0 {
		return nil, false
	}
	merged = prependTrace(merged, "dispatch:"+strings.Join(keys, "|"))
	if len(keys) > 1 {
		merged = downgradeResponsesConfidence(merged, responseConfidenceHeuristic)
	}
	return merged, true
}

func (r *helperResolver) resolveFuncByKey(key string) ([]detectedResponse, bool) {
	if cached, ok := r.cache[key]; ok {
		return cached, len(cached) > 0
	}
	decl, ok := r.declByKey[key]
	if !ok || decl.fn == nil || decl.fn.Body == nil || decl.pkg == nil || decl.pkg.TypesInfo == nil {
		return nil, false
	}
	if r.inStack[key] {
		return nil, false
	}
	r.inStack[key] = true
	defer delete(r.inStack, key)

	found := []detectedResponse{}
	ast.Inspect(decl.fn.Body, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, expr := range ret.Results {
			call, ok := expr.(*ast.CallExpr)
			if !ok {
				continue
			}
			if detected, ok := parseResponseCall(call, decl.pkg.TypesInfo, r); ok {
				found = append(found, detected...)
			}
		}
		return true
	})
	r.cache[key] = dedupeDetectedResponses(found)
	return r.cache[key], len(r.cache[key]) > 0
}

func dedupeDetectedResponses(in []detectedResponse) []detectedResponse {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]detectedResponse, 0, len(in))
	for _, item := range in {
		key := strconv.Itoa(item.status) + "|" + item.typ + "|" + item.contentType + "|" + normalizeResponseConfidence(item.confidence) + "|" + traceKey(item.trace) + "|" + headersKey(item.headers)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func appendDetectedResponses(existing []detectedResponse, detected []detectedResponse) []detectedResponse {
	if len(detected) == 0 {
		return existing
	}
	seen := map[string]struct{}{}
	for _, item := range existing {
		seen[detectedResponseKey(item)] = struct{}{}
	}
	for _, item := range detected {
		key := detectedResponseKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		existing = append(existing, item)
	}
	return existing
}

func detectedResponseKey(item detectedResponse) string {
	return strconv.Itoa(item.status) + "|" + item.typ + "|" + item.contentType + "|" + normalizeResponseConfidence(item.confidence) + "|" + traceKey(item.trace) + "|" + headersKey(item.headers)
}

func toResponseTypeReports(in []detectedResponse) []ResponseTypeReport {
	if len(in) == 0 {
		return nil
	}
	deduped := dedupeDetectedResponses(in)
	ambiguousStatusContent := map[string]bool{}
	typeCountByStatusContent := map[string]map[string]struct{}{}
	for _, item := range deduped {
		key := strconv.Itoa(item.status) + "|" + item.contentType
		if _, ok := typeCountByStatusContent[key]; !ok {
			typeCountByStatusContent[key] = map[string]struct{}{}
		}
		typeCountByStatusContent[key][item.typ] = struct{}{}
	}
	for key, typesByKey := range typeCountByStatusContent {
		if len(typesByKey) > 1 {
			ambiguousStatusContent[key] = true
		}
	}

	out := make([]ResponseTypeReport, 0, len(deduped))
	for _, item := range deduped {
		confidence := normalizeResponseConfidence(item.confidence)
		if ambiguousStatusContent[strconv.Itoa(item.status)+"|"+item.contentType] {
			confidence = responseConfidenceHeuristic
		}
		out = append(out, ResponseTypeReport{
			Status:      item.status,
			Type:        item.typ,
			ContentType: item.contentType,
			Description: item.description,
			Confidence:  confidence,
			Trace:       append([]string(nil), item.trace...),
			Headers:     cloneResponseHeaderReportMap(item.headers),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status != out[j].Status {
			return out[i].Status < out[j].Status
		}
		if out[i].ContentType != out[j].ContentType {
			return out[i].ContentType < out[j].ContentType
		}
		return out[i].Type < out[j].Type
	})
	return out
}

func cloneResponseHeaderReportMap(in map[string]ResponseHeaderReport) map[string]ResponseHeaderReport {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]ResponseHeaderReport, len(in))
	for name, header := range in {
		copyHeader := header
		if len(header.Trace) > 0 {
			copyHeader.Trace = append([]string(nil), header.Trace...)
		}
		out[name] = copyHeader
	}
	return out
}

func headersKey(headers map[string]ResponseHeaderReport) string {
	if len(headers) == 0 {
		return ""
	}
	names := make([]string, 0, len(headers))
	for name := range headers {
		names = append(names, strings.TrimSpace(name))
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		h := headers[name]
		parts = append(parts, name+"|"+h.Type+"|"+h.Description+"|"+normalizeInferenceConfidence(h.Confidence)+"|"+traceKey(h.Trace))
	}
	return strings.Join(parts, "||")
}

func traceKey(trace []string) string {
	if len(trace) == 0 {
		return ""
	}
	return strings.Join(trace, "->")
}

func prependTrace(in []detectedResponse, step string) []detectedResponse {
	step = strings.TrimSpace(step)
	if len(in) == 0 || step == "" {
		return in
	}
	out := make([]detectedResponse, 0, len(in))
	for _, item := range in {
		trace := make([]string, 0, len(item.trace)+1)
		trace = append(trace, step)
		trace = append(trace, item.trace...)
		item.trace = trace
		out = append(out, item)
	}
	return out
}

func normalizeResponseConfidence(c string) string {
	switch strings.TrimSpace(c) {
	case responseConfidenceExact, responseConfidenceInferred, responseConfidenceHeuristic:
		return c
	default:
		return responseConfidenceExact
	}
}

func normalizeInferenceConfidence(c string) string {
	return normalizeResponseConfidence(c)
}

func downgradeResponsesConfidence(in []detectedResponse, minConfidence string) []detectedResponse {
	if len(in) == 0 {
		return nil
	}
	out := make([]detectedResponse, 0, len(in))
	for _, item := range in {
		item.confidence = minResponseConfidence(item.confidence, minConfidence)
		out = append(out, item)
	}
	return out
}

func minResponseConfidence(current, floor string) string {
	curRank := responseConfidenceRank(normalizeResponseConfidence(current))
	floorRank := responseConfidenceRank(normalizeResponseConfidence(floor))
	if curRank >= floorRank {
		return normalizeResponseConfidence(current)
	}
	return normalizeResponseConfidence(floor)
}

func responseConfidenceRank(c string) int {
	switch normalizeResponseConfidence(c) {
	case responseConfidenceExact:
		return 3
	case responseConfidenceInferred:
		return 2
	case responseConfidenceHeuristic:
		return 1
	default:
		return 0
	}
}

func statusCodeFromExpr(expr ast.Expr, info *types.Info) (int, bool) {
	switch v := expr.(type) {
	case *ast.BasicLit:
		if v.Kind != token.INT {
			return 0, false
		}
		n, err := strconv.Atoi(v.Value)
		return n, err == nil
	case *ast.SelectorExpr:
		tv := info.Types[v]
		if tv.Value == nil {
			return 0, false
		}
		n, ok := constantInt(tv.Value.String())
		return n, ok
	default:
		tv := info.Types[expr]
		if tv.Value == nil {
			return 0, false
		}
		n, ok := constantInt(tv.Value.String())
		return n, ok
	}
}

func constantInt(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil
}

func typeOfExprWithConfidence(expr ast.Expr, info *types.Info, vars map[string]types.Type) (types.Type, string) {
	switch v := expr.(type) {
	case *ast.Ident:
		if t, ok := vars[v.Name]; ok {
			return t, responseConfidenceInferred
		}
		return info.TypeOf(v), responseConfidenceInferred
	case *ast.UnaryExpr:
		if v.Op == token.AND {
			t, c := typeOfExprWithConfidence(v.X, info, vars)
			if t != nil {
				return types.NewPointer(t), c
			}
		}
		return info.TypeOf(v), responseConfidenceInferred
	default:
		return info.TypeOf(expr), responseConfidenceExact
	}
}

func canonicalTypeName(t types.Type) string {
	if t == nil {
		return ""
	}
	s := types.TypeString(t, func(p *types.Package) string {
		if p == nil {
			return ""
		}
		return p.Name()
	})
	return strings.TrimPrefix(s, "*")
}

func literalString(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}
