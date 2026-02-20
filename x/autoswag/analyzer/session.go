package analyzer

import (
	"fmt"
	"go/ast"
	"go/types"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

// Session keeps one loaded package universe in memory and supports incremental updates.
type Session struct {
	opts              Options
	cfg               *packages.Config
	patterns          []string
	scope             string
	loadDeps          bool
	explicitOnly      bool
	explicitScope     string
	workspacePrefixes []string

	loaded   bool
	roots    map[string]*packages.Package
	all      map[string]*packages.Package
	scoped   map[string]struct{}
	fileToPk map[string]map[string]struct{}
	dirToPk  map[string]map[string]struct{}
	revDeps  map[string]map[string]struct{}

	declByKey     map[string]helperFuncDecl
	declKeysByPkg map[string]map[string]struct{}
	sourceByInfo  map[*types.Info]diagnosticSource
	sourceInfoPkg map[string]map[*types.Info]struct{}

	funcCalls      map[string][]string
	funcBindings   map[string][]providedBinding
	funcHasUSNew   map[string]bool
	funcPkg        map[string]string
	funcKeysByPkg  map[string]map[string]struct{}
	reportByPkg    map[string]PackageReport
	diagnosticsPkg map[string][]AnalyzerDiagnostic
}

func NewSession(opts Options) (*Session, error) {
	dir := strings.TrimSpace(opts.Dir)
	patterns := opts.Patterns
	if len(patterns) == 0 {
		patterns = []string{"."}
	}
	loadDeps := opts.LoadDeps
	if opts.ExplicitOnly {
		loadDeps = false
	}
	if !opts.ExplicitOnly && !loadDeps && normalizeIndexScope(opts.IndexScope) == "all" {
		loadDeps = true
	}
	mode := packages.NeedName | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedSyntax | packages.NeedFiles | packages.NeedImports
	if loadDeps {
		mode |= packages.NeedDeps
	}
	cfg := &packages.Config{
		Mode: mode,
		Dir:  dir,
	}
	if strings.TrimSpace(opts.Tags) != "" {
		cfg.BuildFlags = []string{"-tags=" + strings.TrimSpace(opts.Tags)}
	}
	if len(opts.Overlay) > 0 {
		cfg.Overlay = map[string][]byte{}
		for k, v := range opts.Overlay {
			cfg.Overlay[k] = append([]byte(nil), v...)
		}
	}
	return &Session{
		opts:           opts,
		cfg:            cfg,
		patterns:       append([]string(nil), patterns...),
		scope:          normalizeIndexScope(opts.IndexScope),
		loadDeps:       loadDeps,
		explicitOnly:   opts.ExplicitOnly,
		explicitScope:  normalizeExplicitScope(opts.ExplicitScope),
		roots:          map[string]*packages.Package{},
		all:            map[string]*packages.Package{},
		scoped:         map[string]struct{}{},
		fileToPk:       map[string]map[string]struct{}{},
		dirToPk:        map[string]map[string]struct{}{},
		revDeps:        map[string]map[string]struct{}{},
		declByKey:      map[string]helperFuncDecl{},
		declKeysByPkg:  map[string]map[string]struct{}{},
		sourceByInfo:   map[*types.Info]diagnosticSource{},
		sourceInfoPkg:  map[string]map[*types.Info]struct{}{},
		funcCalls:      map[string][]string{},
		funcBindings:   map[string][]providedBinding{},
		funcHasUSNew:   map[string]bool{},
		funcPkg:        map[string]string{},
		funcKeysByPkg:  map[string]map[string]struct{}{},
		reportByPkg:    map[string]PackageReport{},
		diagnosticsPkg: map[string][]AnalyzerDiagnostic{},
	}, nil
}

func (s *Session) Analyze() (*Report, error) {
	started := time.Now()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	report, err := s.analyzeRootPackages(s.rootPkgPaths())
	if err == nil {
		s.progress(fmt.Sprintf("analyze run took %s (%s)", time.Since(started).Round(time.Millisecond), heapStatsString()))
	}
	return report, err
}

func (s *Session) AnalyzeChangedFiles(files []string) (*Report, error) {
	started := time.Now()
	if err := s.ensureLoaded(); err != nil {
		return nil, err
	}
	changed := normalizeFileList(files)
	if len(changed) == 0 {
		return s.currentReport(), nil
	}
	s.progress(fmt.Sprintf("watch change set: %s", summarizeList(changed, 12)))
	fullReload, invalidated := s.computeInvalidatedPackages(changed)
	if err := s.applyOverlay(changed); err != nil {
		return nil, err
	}
	if fullReload {
		s.progress("invalidated: full reload")
		if err := s.loadUniverse(s.patterns, true, nil); err != nil {
			return nil, err
		}
		report, err := s.analyzeRootPackages(s.rootPkgPaths())
		if err == nil {
			s.progress(fmt.Sprintf("watch rerun took %s (%s)", time.Since(started).Round(time.Millisecond), heapStatsString()))
		}
		return report, err
	}
	if len(invalidated) == 0 {
		return s.currentReport(), nil
	}
	s.progress(fmt.Sprintf("invalidated packages: %s", summarizeList(keysOfSet(invalidated), 12)))
	if err := s.loadUniverse(keysOfSet(invalidated), false, invalidated); err != nil {
		return nil, err
	}
	roots := s.invalidatedRoots(invalidated)
	if s.explicitOnly {
		roots = s.rootPkgPaths()
	}
	if len(roots) == 0 {
		return s.currentReport(), nil
	}
	s.progress(fmt.Sprintf("reanalyzing root packages: %s", summarizeList(roots, 12)))
	report, err := s.analyzeRootPackages(roots)
	if err == nil {
		s.progress(fmt.Sprintf("watch rerun took %s (%s)", time.Since(started).Round(time.Millisecond), heapStatsString()))
	}
	return report, err
}

func (s *Session) ensureLoaded() error {
	if s.loaded {
		return nil
	}
	return s.loadUniverse(s.patterns, true, nil)
}

func (s *Session) progress(msg string) {
	if s.opts.Progress != nil {
		s.opts.Progress(msg)
	}
}

func (s *Session) loadUniverse(patterns []string, full bool, invalidated map[string]struct{}) error {
	started := time.Now()
	s.progress("loading packages")
	if s.loadDeps {
		s.progress("load mode: deep (deps)")
	} else {
		s.progress("load mode: shallow (no deps)")
	}
	if len(patterns) > 0 {
		s.progress(fmt.Sprintf("load patterns: %s", summarizeList(patterns, 12)))
	}
	pkgs, err := packages.Load(s.cfg, patterns...)
	if err != nil {
		return err
	}
	if packages.PrintErrors(pkgs) > 0 {
		return fmt.Errorf("failed to load packages")
	}
	if full {
		s.roots = map[string]*packages.Package{}
		for _, pkg := range pkgs {
			if pkg == nil || strings.TrimSpace(pkg.PkgPath) == "" {
				continue
			}
			s.roots[pkg.PkgPath] = pkg
		}
	} else {
		loadedByPath := map[string]*packages.Package{}
		for _, p := range collectAllPackages(pkgs) {
			if p == nil || strings.TrimSpace(p.PkgPath) == "" {
				continue
			}
			loadedByPath[p.PkgPath] = p
		}
		for path, rp := range s.roots {
			if next, ok := loadedByPath[path]; ok {
				s.roots[path] = next
			} else if rp == nil {
				delete(s.roots, path)
			}
		}
	}

	s.rebuildUniverseFromRoots()
	s.rebuildGraphMaps()
	prevScoped := s.scoped
	s.scoped = s.computeScopedSet()
	s.progress(fmt.Sprintf("expanded dependency graph to %d packages (scope=%s)", len(s.scoped), s.scope))
	if full {
		s.clearIndexState()
		s.reindexPackages(s.scoped)
	} else {
		toReindex := map[string]struct{}{}
		for p := range invalidated {
			if _, ok := s.scoped[p]; ok {
				toReindex[p] = struct{}{}
			}
		}
		for p := range prevScoped {
			if _, ok := s.scoped[p]; !ok {
				toReindex[p] = struct{}{}
			}
		}
		for p := range s.scoped {
			if _, ok := prevScoped[p]; !ok {
				toReindex[p] = struct{}{}
			}
		}
		if len(toReindex) > 0 {
			s.progress(fmt.Sprintf("reindexing packages: %s", summarizeList(keysOfSet(toReindex), 12)))
		}
		s.reindexPackages(toReindex)
	}
	s.loaded = true
	s.progress(fmt.Sprintf("loaded %d root packages", len(s.roots)))
	s.progress(fmt.Sprintf("watchable files mapped: %d", len(s.fileToPk)))
	s.progress(fmt.Sprintf("load+index phase took %s (%s)", time.Since(started).Round(time.Millisecond), heapStatsString()))
	return nil
}

func (s *Session) rebuildUniverseFromRoots() {
	s.all = map[string]*packages.Package{}
	roots := s.rootPackages()
	if s.explicitOnly {
		switch s.explicitScope {
		case "imports":
			for _, p := range roots {
				if p == nil || strings.TrimSpace(p.PkgPath) == "" {
					continue
				}
				s.all[p.PkgPath] = p
				for impPath, imp := range p.Imports {
					if imp == nil || strings.TrimSpace(impPath) == "" {
						continue
					}
					s.all[impPath] = imp
				}
			}
			return
		case "workspace":
			s.workspacePrefixes = workspacePrefixesFromRoots(roots)
			for _, p := range collectAllPackages(roots) {
				if p == nil || strings.TrimSpace(p.PkgPath) == "" {
					continue
				}
				if !hasWorkspacePrefix(p.PkgPath, s.workspacePrefixes) {
					continue
				}
				s.all[p.PkgPath] = p
			}
			return
		case "all":
			for _, p := range collectAllPackages(roots) {
				if p == nil || strings.TrimSpace(p.PkgPath) == "" {
					continue
				}
				s.all[p.PkgPath] = p
			}
			return
		default:
			// roots
		}
	}
	if s.scope == "roots" {
		for _, p := range roots {
			if p == nil || strings.TrimSpace(p.PkgPath) == "" {
				continue
			}
			s.all[p.PkgPath] = p
		}
		return
	}
	s.workspacePrefixes = workspacePrefixesFromRoots(roots)
	for _, p := range collectAllPackages(roots) {
		if p == nil || strings.TrimSpace(p.PkgPath) == "" {
			continue
		}
		if !hasWorkspacePrefix(p.PkgPath, s.workspacePrefixes) {
			continue
		}
		s.all[p.PkgPath] = p
	}
}

func (s *Session) rebuildGraphMaps() {
	s.fileToPk = map[string]map[string]struct{}{}
	s.dirToPk = map[string]map[string]struct{}{}
	s.revDeps = map[string]map[string]struct{}{}
	for pkgPath, pkg := range s.all {
		if pkg == nil {
			continue
		}
		for _, f := range pkg.GoFiles {
			abs := cleanAbsOrSelf(f)
			if abs == "" {
				continue
			}
			if _, ok := s.fileToPk[abs]; !ok {
				s.fileToPk[abs] = map[string]struct{}{}
			}
			s.fileToPk[abs][pkgPath] = struct{}{}
			dir := filepath.Dir(abs)
			if _, ok := s.dirToPk[dir]; !ok {
				s.dirToPk[dir] = map[string]struct{}{}
			}
			s.dirToPk[dir][pkgPath] = struct{}{}
		}
		for impPath := range pkg.Imports {
			if strings.TrimSpace(impPath) == "" {
				continue
			}
			if _, ok := s.revDeps[impPath]; !ok {
				s.revDeps[impPath] = map[string]struct{}{}
			}
			s.revDeps[impPath][pkgPath] = struct{}{}
		}
	}
}

func (s *Session) computeScopedSet() map[string]struct{} {
	if s.explicitOnly {
		out := map[string]struct{}{}
		for path := range s.all {
			out[path] = struct{}{}
		}
		return out
	}
	roots := s.rootPackages()
	all := s.allPackages()
	scoped := selectPackagesByScope(roots, all, s.scope)
	out := map[string]struct{}{}
	for _, p := range scoped {
		if p == nil {
			continue
		}
		out[p.PkgPath] = struct{}{}
	}
	return out
}

func (s *Session) reindexPackages(pkgs map[string]struct{}) {
	for pkgPath := range pkgs {
		s.removePackageIndex(pkgPath)
		if _, ok := s.scoped[pkgPath]; !ok {
			continue
		}
		pkg := s.all[pkgPath]
		if pkg == nil || pkg.TypesInfo == nil {
			continue
		}
		if pkg.TypesInfo != nil && pkg.Fset != nil {
			s.sourceByInfo[pkg.TypesInfo] = diagnosticSource{pkgPath: pkgPath, fset: pkg.Fset}
			if _, ok := s.sourceInfoPkg[pkgPath]; !ok {
				s.sourceInfoPkg[pkgPath] = map[*types.Info]struct{}{}
			}
			s.sourceInfoPkg[pkgPath][pkg.TypesInfo] = struct{}{}
		}
		s.addPackageDeclIndex(pkg)
		s.addPackageFunctionIndex(pkg)
	}
}

func (s *Session) removePackageIndex(pkgPath string) {
	if keys, ok := s.declKeysByPkg[pkgPath]; ok {
		for key := range keys {
			delete(s.declByKey, key)
		}
	}
	delete(s.declKeysByPkg, pkgPath)
	if infos, ok := s.sourceInfoPkg[pkgPath]; ok {
		for info := range infos {
			delete(s.sourceByInfo, info)
		}
	}
	delete(s.sourceInfoPkg, pkgPath)
	if fnKeys, ok := s.funcKeysByPkg[pkgPath]; ok {
		for key := range fnKeys {
			delete(s.funcCalls, key)
			delete(s.funcBindings, key)
			delete(s.funcHasUSNew, key)
			delete(s.funcPkg, key)
		}
	}
	delete(s.funcKeysByPkg, pkgPath)
	delete(s.reportByPkg, pkgPath)
	delete(s.diagnosticsPkg, pkgPath)
}

func (s *Session) clearIndexState() {
	s.declByKey = map[string]helperFuncDecl{}
	s.declKeysByPkg = map[string]map[string]struct{}{}
	s.sourceByInfo = map[*types.Info]diagnosticSource{}
	s.sourceInfoPkg = map[string]map[*types.Info]struct{}{}
	s.funcCalls = map[string][]string{}
	s.funcBindings = map[string][]providedBinding{}
	s.funcHasUSNew = map[string]bool{}
	s.funcPkg = map[string]string{}
	s.funcKeysByPkg = map[string]map[string]struct{}{}
	s.reportByPkg = map[string]PackageReport{}
	s.diagnosticsPkg = map[string][]AnalyzerDiagnostic{}
}

func (s *Session) addPackageDeclIndex(pkg *packages.Package) {
	keys := map[string]struct{}{}
	for key, decl := range buildHelperDeclIndexForPackage(pkg) {
		s.declByKey[key] = decl
		keys[key] = struct{}{}
	}
	if len(keys) > 0 {
		s.declKeysByPkg[pkg.PkgPath] = keys
	}
}

func (s *Session) addPackageFunctionIndex(pkg *packages.Package) {
	keys := map[string]struct{}{}
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Body == nil {
				continue
			}
			obj := pkg.TypesInfo.Defs[fn.Name]
			f, ok := obj.(*types.Func)
			if !ok || f == nil {
				continue
			}
			key := strings.TrimSpace(f.FullName())
			if key == "" {
				continue
			}
			keys[key] = struct{}{}
			s.funcPkg[key] = pkg.PkgPath
			calls := []string{}
			bindings := []providedBinding{}
			hasUSNew := false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if called, ok := resolveCallExprFuncKey(call, pkg.TypesInfo); ok {
					calls = append(calls, called)
				}
				if isDIProvideCall(call, pkg.TypesInfo) {
					if binding, ok := providedBindingFromProvideCall(call, pkg.TypesInfo); ok {
						bindings = append(bindings, binding)
					}
				}
				if isUSNewCall(call, pkg.TypesInfo) {
					hasUSNew = true
				}
				return true
			})
			s.funcCalls[key] = dedupeStrings(calls)
			s.funcBindings[key] = dedupeProvidedBindings(bindings)
			s.funcHasUSNew[key] = hasUSNew
		}
	}
	if len(keys) > 0 {
		s.funcKeysByPkg[pkg.PkgPath] = keys
	}
}

func (s *Session) applyOverlay(changed []string) error {
	if s.cfg.Overlay == nil {
		s.cfg.Overlay = map[string][]byte{}
	}
	for _, f := range changed {
		name := filepath.Base(f)
		if !isSessionSourceFile(name) {
			continue
		}
		b, err := os.ReadFile(f)
		if err != nil {
			delete(s.cfg.Overlay, f)
			continue
		}
		s.cfg.Overlay[f] = b
	}
	return nil
}

func (s *Session) computeInvalidatedPackages(changed []string) (bool, map[string]struct{}) {
	base := map[string]struct{}{}
	full := false
	for _, f := range changed {
		name := filepath.Base(f)
		if name == "go.mod" || name == "go.sum" {
			full = true
			break
		}
		pkgs := map[string]struct{}{}
		for p := range s.fileToPk[f] {
			pkgs[p] = struct{}{}
		}
		for p := range s.dirToPk[filepath.Dir(f)] {
			pkgs[p] = struct{}{}
		}
		if len(pkgs) == 0 {
			full = true
			break
		}
		for p := range pkgs {
			base[p] = struct{}{}
		}
	}
	if full {
		return true, nil
	}
	if s.explicitOnly {
		return false, base
	}
	return false, s.expandReverseDeps(base)
}

func (s *Session) expandReverseDeps(base map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	queue := make([]string, 0, len(base))
	for p := range base {
		queue = append(queue, p)
		out[p] = struct{}{}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for dep := range s.revDeps[cur] {
			if _, ok := out[dep]; ok {
				continue
			}
			out[dep] = struct{}{}
			queue = append(queue, dep)
		}
	}
	return out
}

func (s *Session) invalidatedRoots(invalidated map[string]struct{}) []string {
	out := []string{}
	for path := range s.roots {
		if _, ok := invalidated[path]; ok {
			out = append(out, path)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Session) analyzeRootPackages(rootPaths []string) (*Report, error) {
	started := time.Now()
	scopedPkgs := s.scopedPackages()
	rootPkgs := s.rootPackages()
	provided := []providedBinding(nil)
	if !s.explicitOnly {
		provided = s.providedBindings()
	}
	decls := s.declByKey
	if !s.explicitOnly && (s.scope == "referenced" || s.scope == "workspace" || s.scope == "roots") {
		decls = buildReferencedHelperDeclIndex(scopedPkgs, rootPkgs, decls, provided)
	}
	s.progress(fmt.Sprintf("indexed %d helper functions and %d DI bindings", len(decls), len(provided)))
	for _, path := range rootPaths {
		pkg := s.roots[path]
		if pkg == nil {
			continue
		}
		pkgStarted := time.Now()
		s.progress("analyzing package: " + pkg.PkgPath)
		out := PackageReport{
			Path:    pkg.PkgPath,
			Imports: collectPackageImports(pkg),
		}
		var packageDiags []AnalyzerDiagnostic
		cacheHit := false
		fingerprint, fpErr := packageFingerprint(pkg, s.scope, s.opts.Tags, s.opts.StrictDI, s.opts.ToolVersion)
		if fpErr == nil && s.opts.PackageCacheLoad != nil {
			if cached, ok := s.opts.PackageCacheLoad(pkg.PkgPath, fingerprint); ok && cached != nil {
				cacheHit = true
				out = cached.Package
				if strings.TrimSpace(out.Path) == "" {
					out.Path = pkg.PkgPath
				}
				if len(out.Imports) == 0 {
					out.Imports = collectPackageImports(pkg)
				}
				packageDiags = append(packageDiags, cached.Diagnostics...)
				s.progress("package cache hit: " + pkg.PkgPath)
			} else {
				s.progress("package cache miss: " + pkg.PkgPath)
			}
		}
		if !cacheHit {
			resolver := newHelperResolver(decls, provided, pkg.PkgPath, s.opts.StrictDI, s.explicitOnly, s.sourceByInfo)
			for _, file := range pkg.Syntax {
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Body == nil || fn.Type == nil || fn.Type.Params == nil {
						continue
					}
					if !isFiberCtxHandler(fn, pkg.TypesInfo) {
						if isRouterHandleMethod(fn, pkg.TypesInfo) {
							routes, inlineHandlers := extractRouteBindings(pkg, fn, resolver)
							out.Routes = append(out.Routes, routes...)
							out.Handlers = append(out.Handlers, inlineHandlers...)
						}
						continue
					}
					out.Handlers = append(out.Handlers, analyzeFunc(pkg, fn, resolver))
				}
			}
			if resolver != nil && len(resolver.diagnostics) > 0 {
				annotateDiagnosticsRoutes(resolver.diagnostics, out.Routes)
				packageDiags = append(packageDiags, resolver.diagnostics...)
			}
			if fpErr == nil && s.opts.PackageCacheStore != nil {
				s.opts.PackageCacheStore(pkg.PkgPath, fingerprint, PackageCacheEntry{
					Package:     out,
					Diagnostics: packageDiags,
				})
			}
		}
		sort.Slice(out.Handlers, func(i, j int) bool {
			return out.Handlers[i].Name < out.Handlers[j].Name
		})
		sort.Slice(out.Routes, func(i, j int) bool {
			if out.Routes[i].Path == out.Routes[j].Path {
				return out.Routes[i].Method < out.Routes[j].Method
			}
			return out.Routes[i].Path < out.Routes[j].Path
		})
		s.reportByPkg[path] = out
		s.diagnosticsPkg[path] = packageDiags
		s.progress(fmt.Sprintf("analyzed package: %s (%s)", pkg.PkgPath, time.Since(pkgStarted).Round(time.Millisecond)))
	}
	report := s.currentReport()
	s.progress(fmt.Sprintf("analysis complete: %d package reports, %d diagnostics", len(report.Packages), len(report.Diagnostics)))
	s.progress(fmt.Sprintf("analyze phase took %s (%s)", time.Since(started).Round(time.Millisecond), heapStatsString()))
	if s.opts.StrictDI {
		for _, d := range report.Diagnostics {
			if d.Severity == "error" {
				return report, fmt.Errorf("strict-di: %s: %s", d.Code, d.Message)
			}
		}
	}
	return report, nil
}

func (s *Session) providedBindings() []providedBinding {
	seeds := map[string]struct{}{}
	for key, has := range s.funcHasUSNew {
		if !has {
			continue
		}
		if _, ok := s.roots[s.funcPkg[key]]; ok {
			seeds[key] = struct{}{}
		}
	}
	if len(seeds) == 0 {
		out := []providedBinding{}
		for key, bindings := range s.funcBindings {
			if _, ok := s.scoped[s.funcPkg[key]]; !ok {
				continue
			}
			out = append(out, bindings...)
		}
		return dedupeProvidedBindings(out)
	}
	queue := make([]string, 0, len(seeds))
	seen := map[string]struct{}{}
	for key := range seeds {
		queue = append(queue, key)
	}
	out := []providedBinding{}
	for len(queue) > 0 {
		key := queue[0]
		queue = queue[1:]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, s.funcBindings[key]...)
		for _, next := range s.funcCalls[key] {
			if _, ok := seen[next]; ok {
				continue
			}
			queue = append(queue, next)
		}
	}
	return dedupeProvidedBindings(out)
}

func (s *Session) currentReport() *Report {
	paths := s.rootPkgPaths()
	var graph *DependencyGraph
	if !s.explicitOnly {
		graph = buildDependencyGraph(s.providedBindings())
	}
	report := &Report{
		Packages:        make([]PackageReport, 0, len(paths)),
		Diagnostics:     []AnalyzerDiagnostic{},
		DependencyGraph: graph,
	}
	for _, path := range paths {
		if p, ok := s.reportByPkg[path]; ok {
			report.Packages = append(report.Packages, p)
		}
		report.Diagnostics = append(report.Diagnostics, s.diagnosticsPkg[path]...)
	}
	sort.Slice(report.Packages, func(i, j int) bool {
		return report.Packages[i].Path < report.Packages[j].Path
	})
	sort.Slice(report.Diagnostics, func(i, j int) bool {
		if report.Diagnostics[i].Package == report.Diagnostics[j].Package {
			if report.Diagnostics[i].Code == report.Diagnostics[j].Code {
				return report.Diagnostics[i].Message < report.Diagnostics[j].Message
			}
			return report.Diagnostics[i].Code < report.Diagnostics[j].Code
		}
		return report.Diagnostics[i].Package < report.Diagnostics[j].Package
	})
	return report
}

func (s *Session) rootPackages() []*packages.Package {
	out := make([]*packages.Package, 0, len(s.roots))
	for _, p := range s.roots {
		if p != nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PkgPath < out[j].PkgPath
	})
	return out
}

func (s *Session) allPackages() []*packages.Package {
	out := make([]*packages.Package, 0, len(s.all))
	for _, p := range s.all {
		if p != nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PkgPath < out[j].PkgPath
	})
	return out
}

func (s *Session) scopedPackages() []*packages.Package {
	out := make([]*packages.Package, 0, len(s.scoped))
	for path := range s.scoped {
		if p := s.all[path]; p != nil {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PkgPath < out[j].PkgPath
	})
	return out
}

func (s *Session) rootPkgPaths() []string {
	out := make([]string, 0, len(s.roots))
	for path := range s.roots {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func buildHelperDeclIndexForPackage(pkg *packages.Package) map[string]helperFuncDecl {
	index := map[string]helperFuncDecl{}
	if pkg == nil || pkg.TypesInfo == nil {
		return index
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
	return index
}

func cleanAbsOrSelf(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func normalizeFileList(files []string) []string {
	set := map[string]struct{}{}
	out := make([]string, 0, len(files))
	for _, f := range files {
		c := cleanAbsOrSelf(f)
		if c == "" {
			continue
		}
		if _, ok := set[c]; ok {
			continue
		}
		set[c] = struct{}{}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func keysOfSet(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func dedupeStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func isSessionSourceFile(name string) bool {
	name = strings.TrimSpace(name)
	return name == "go.mod" || name == "go.sum" || strings.HasSuffix(name, ".go")
}

func hasWorkspacePrefix(pkgPath string, prefixes []string) bool {
	pkgPath = strings.TrimSpace(pkgPath)
	if pkgPath == "" {
		return false
	}
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if pkgPath == prefix || strings.HasPrefix(pkgPath, prefix+"/") {
			return true
		}
	}
	return false
}

func normalizeExplicitScope(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case "roots", "imports", "workspace", "all":
		return strings.ToLower(strings.TrimSpace(scope))
	default:
		return "roots"
	}
}

func summarizeList(items []string, max int) string {
	if len(items) == 0 {
		return "(none)"
	}
	if max <= 0 {
		max = 1
	}
	if len(items) <= max {
		return strings.Join(items, ",")
	}
	head := strings.Join(items[:max], ",")
	return head + ",...+" + strconv.Itoa(len(items)-max)
}

func heapStatsString() string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	mb := func(v uint64) string {
		return fmt.Sprintf("%.1fMB", float64(v)/1024.0/1024.0)
	}
	return "heap_alloc=" + mb(ms.HeapAlloc) + ",heap_inuse=" + mb(ms.HeapInuse) + ",num_gc=" + strconv.FormatUint(uint64(ms.NumGC), 10)
}
