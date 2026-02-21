package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer"
	"github.com/fsnotify/fsnotify"
)

func main() {
	startedAt := time.Now()

	var dir string
	var pattern string
	var emitHook bool
	var hookPackage string
	var hookName string
	var out string
	var tags string
	var exactOnly bool
	var strictDI bool
	explicitOnly := true
	explicitScope := "imports"
	var diagOut string
	var graphOut string
	var verbose bool
	var deep bool
	var indexScope string
	var diagOnly bool
	var diagGrouped bool
	var lint bool
	var lintSeverity string
	var disableCommentDetection bool
	var disableDirectiveDetection bool
	var cacheDir string
	var noCache bool
	var cachePrune bool
	var watch bool
	var watchInterval time.Duration
	flag.StringVar(&dir, "dir", ".", "working directory")
	flag.StringVar(&pattern, "patterns", ".", "comma-separated go package patterns")
	flag.BoolVar(&emitHook, "emit-hook", false, "emit autoswag WithCustomize hook source instead of JSON report")
	flag.StringVar(&hookPackage, "hook-package", "autoswaggen", "package name for generated hook source")
	flag.StringVar(&hookName, "emit-hook-name", "AutoSwagGenerator", "function name for generated hook source")
	flag.StringVar(&out, "out", "", "write output to file path (default: stdout)")
	flag.StringVar(&tags, "tags", "", "build tags for package loading (comma-separated or go -tags expression)")
	flag.BoolVar(&exactOnly, "exact-only", false, "when -emit-hook is enabled, include only exact-confidence detections")
	flag.BoolVar(&strictDI, "strict-di", false, "treat ambiguous DI interface dispatch as an error")
	flag.BoolVar(&explicitOnly, "explicit-only", true, "limit response inference to explicit c.* and concrete helper methods (*.Method(c,...)); disables DI/deep helper inference")
	flag.StringVar(&explicitScope, "explicit-scope", "imports", "explicit-only package scope: roots|imports|workspace|all|auto")
	flag.StringVar(&diagOut, "diag-out", "", "write analyzer diagnostics JSON to file path")
	flag.StringVar(&graphOut, "graph-out", "", "write dependency graph JSON to file path")
	flag.BoolVar(&verbose, "verbose", false, "print analyzer progress logs to stderr")
	flag.BoolVar(&deep, "deep", true, "enable deep indexing (equivalent to --index-scope all when index-scope is not set)")
	flag.StringVar(&indexScope, "index-scope", "", "index scope: auto|workspace|roots|referenced|all (overrides --deep)")
	flag.BoolVar(&diagOnly, "diag-only", false, "print diagnostics only (human-readable), do not emit JSON report/hook payload")
	flag.BoolVar(&diagGrouped, "diag-grouped", true, "group diagnostics by route/package in -diag-only output")
	flag.BoolVar(&lint, "lint", false, "run analyzer lint checks and exit non-zero when diagnostics meet severity threshold")
	flag.StringVar(&lintSeverity, "lint-severity", "warning", "lint failure threshold: warning|error")
	flag.BoolVar(&disableCommentDetection, "disable-comment-detection", false, "disable non-directive comment-based detection (e.g. nearby description comments)")
	flag.BoolVar(&disableDirectiveDetection, "disable-directive-detection", false, "disable @autoswag:* directive detection")
	flag.StringVar(&cacheDir, "cache-dir", defaultCacheDir(), "directory for analysis cache")
	flag.BoolVar(&noCache, "no-cache", false, "disable reading/writing analysis cache")
	flag.BoolVar(&cachePrune, "cache-prune", false, "delete cache directory and exit")
	flag.BoolVar(&watch, "watch", false, "watch files and rerun analyze command on changes")
	flag.DurationVar(&watchInterval, "watch-interval", 1*time.Second, "watch polling interval (e.g. 500ms, 2s)")
	flag.Parse()

	cacheRoot := strings.TrimSpace(cacheDir)
	if cachePrune {
		if cacheRoot == "" {
			fmt.Fprintln(os.Stderr, "cache-prune: cache directory is empty")
			os.Exit(1)
		}
		if err := pruneCacheDir(cacheRoot); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		fmt.Printf("cache pruned: %s\n", cacheRoot)
		return
	}

	scope := strings.TrimSpace(indexScope)
	if scope == "" {
		if deep {
			scope = "auto"
		} else {
			scope = "workspace"
		}
	}
	scope = strings.ToLower(strings.TrimSpace(scope))
	if scope == "" {
		scope = "all"
	}

	patterns := []string{"."}
	if strings.TrimSpace(pattern) != "" {
		patterns = strings.Split(pattern, ",")
		for i := range patterns {
			patterns[i] = strings.TrimSpace(patterns[i])
		}
	}
	if watch {
		toolVersion := detectToolVersion()
		cacheEnabled := !noCache && cacheRoot != ""
		watchSessionRecycleHeapMB := watchRecycleHeapMB()
		if watchSessionRecycleHeapMB > 0 {
			debug.SetMemoryLimit(int64(watchSessionRecycleHeapMB * 1024 * 1024))
		}
		autoWatch := scope == "auto"
		watchScopes := []string{scope}
		if autoWatch {
			watchScopes = autoAnalyzeScopes(strictDI, explicitOnly, explicitScope)
			if verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] watch auto scopes: %s\n", time.Now().Format("15:04:05"), strings.Join(watchScopes, " -> "))
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] watch heap guard: recycle=%dMB\n", time.Now().Format("15:04:05"), watchSessionRecycleHeapMB)
			}
		}
		emit := func(report *analyzer.Report) error {
			if report == nil {
				return nil
			}
			if strings.TrimSpace(diagOut) != "" {
				writeDiagnostics(diagOut, report.Diagnostics)
			}
			if strings.TrimSpace(graphOut) != "" {
				writeDependencyGraph(graphOut, report.DependencyGraph)
			}
			if lint {
				violations := filterLintDiagnostics(report.Diagnostics, lintSeverity)
				if len(violations) > 0 {
					printDiagnostics(violations, diagGrouped)
					return fmt.Errorf("lint failed: %d diagnostic(s) at severity >= %s", len(violations), normalizeLintSeverity(lintSeverity))
				}
				if verbose {
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] lint ok (severity>=%s)\n", time.Now().Format("15:04:05"), normalizeLintSeverity(lintSeverity))
				}
				return nil
			}
			if diagOnly {
				printDiagnostics(report.Diagnostics, diagGrouped)
				return nil
			}
			payload, err := buildPayload(report, emitHook, hookPackage, hookName, exactOnly, toolVersion)
			if err != nil {
				return err
			}
			return writePayload(out, payload)
		}
		watchScopeIdx := 0
		newWatchSession := func(scope string) (*analyzer.Session, error) {
			return newAnalyzeSession(dir, patterns, tags, strictDI, explicitOnly, explicitScope, scope, toolVersion, verbose, cacheEnabled, cacheRoot, disableCommentDetection, disableDirectiveDetection)
		}
		var session *analyzer.Session
		recycleWatchSession := func(reason string) error {
			if verbose {
				fmt.Fprintf(
					os.Stderr,
					"[autoswag-analyze %s] recycling watch session (%s, scope=%s, heap_alloc=%dMB)\n",
					time.Now().Format("15:04:05"),
					reason,
					watchScopes[watchScopeIdx],
					heapAllocMB(),
				)
			}
			session = nil
			runtime.GC()
			debug.FreeOSMemory()
			next, err := newWatchSession(watchScopes[watchScopeIdx])
			if err != nil {
				return err
			}
			session = next
			if verbose {
				fmt.Fprintf(
					os.Stderr,
					"[autoswag-analyze %s] watch session recycled (heap_alloc=%dMB)\n",
					time.Now().Format("15:04:05"),
					heapAllocMB(),
				)
			}
			return nil
		}
		session, err := newWatchSession(watchScopes[watchScopeIdx])
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		runWatchAnalyze := func(changed []string, initial bool) (*analyzer.Report, error) {
			for {
				var report *analyzer.Report
				var err error
				if initial {
					report, err = session.Analyze()
				} else {
					report, err = session.AnalyzeChangedFiles(changed)
				}
				if err != nil {
					return report, err
				}
				if !autoWatch || !shouldEscalateAnalyzeReport(report) || watchScopeIdx >= len(watchScopes)-1 {
					return report, nil
				}
				prevScope := watchScopes[watchScopeIdx]
				watchScopeIdx++
				nextScope := watchScopes[watchScopeIdx]
				if verbose {
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] auto escalation requested: %s -> %s\n", time.Now().Format("15:04:05"), prevScope, nextScope)
				}
				session, err = newWatchSession(nextScope)
				if err != nil {
					return nil, err
				}
				initial = true
			}
		}
		report, err := runWatchAnalyze(nil, true)
		if err != nil {
			if report != nil && strings.TrimSpace(diagOut) != "" {
				writeDiagnostics(diagOut, report.Diagnostics)
			}
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := emit(report); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := runWatchMode(cacheRoot, dir, patterns, out, watchInterval, verbose, func(changed []string) error {
			report, err := runWatchAnalyze(changed, false)
			if err != nil {
				if report != nil && strings.TrimSpace(diagOut) != "" {
					writeDiagnostics(diagOut, report.Diagnostics)
				}
				return err
			}
			if err := emit(report); err != nil {
				return err
			}
			if explicitOnly {
				runtime.GC()
				debug.FreeOSMemory()
				if verbose {
					fmt.Fprintf(
						os.Stderr,
						"[autoswag-analyze %s] post-run memory trim (heap_alloc=%dMB)\n",
						time.Now().Format("15:04:05"),
						heapAllocMB(),
					)
				}
			}
			if heapAllocMB() >= watchSessionRecycleHeapMB {
				return recycleWatchSession("heap threshold")
			}
			return nil
		}); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		return
	}
	toolVersion := detectToolVersion()

	var report *analyzer.Report
	var err error
	cacheKey := ""
	cachePath := ""
	cacheEnabled := !noCache && cacheRoot != ""
	if cacheEnabled {
		cacheKey, err = computeCacheKey(dir, patterns, tags, strictDI, scope, toolVersion, disableCommentDetection, disableDirectiveDetection)
		if err == nil && cacheKey != "" {
			cachePath = filepath.Join(cacheRoot, cacheKey+".json")
			if cached, readErr := readCachedReport(cachePath); readErr == nil && cached != nil {
				report = cached
				if verbose {
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] nothing changed (%s)\n", time.Now().Format("15:04:05"), time.Since(startedAt).Round(time.Millisecond))
				}
			} else if verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] cache miss\n", time.Now().Format("15:04:05"))
			}
		}
	}

	if report == nil {
		runAnalyze := func(runScope string) (*analyzer.Report, error) {
			opts := analyzer.Options{
				Dir:           dir,
				Patterns:      patterns,
				Tags:          tags,
				StrictDI:      strictDI,
				DisableCommentDetection: disableCommentDetection,
				DisableDirectiveDetection: disableDirectiveDetection,
				ExplicitOnly:  explicitOnly,
				ExplicitScope: explicitScopeForScope(explicitOnly, explicitScope, runScope),
				IndexScope:    runScope,
				LoadDeps:      shouldLoadDepsForScope(runScope, explicitOnly),
				ToolVersion:   toolVersion,
				Progress: func(message string) {
					if !verbose {
						return
					}
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] %s\n", time.Now().Format("15:04:05"), strings.TrimSpace(message))
				},
			}
			if cacheEnabled {
				opts.PackageCacheLoad = func(pkgPath, fingerprint string) (*analyzer.PackageCacheEntry, bool) {
					entry, loadErr := readCachedPackageEntry(cacheRoot, pkgPath, fingerprint)
					return entry, loadErr == nil && entry != nil
				}
				opts.PackageCacheStore = func(pkgPath, fingerprint string, entry analyzer.PackageCacheEntry) {
					if writeErr := writeCachedPackageEntry(cacheRoot, pkgPath, fingerprint, entry); writeErr != nil && verbose {
						fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] package cache write failed (%s): %v\n", time.Now().Format("15:04:05"), strings.TrimSpace(pkgPath), writeErr)
					}
				}
			}
			return analyzer.Analyze(opts)
		}

		if scope == "auto" {
			autoScopes := autoAnalyzeScopes(strictDI, explicitOnly, explicitScope)
			for i, runScope := range autoScopes {
				if verbose {
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] auto scope pass: %s\n", time.Now().Format("15:04:05"), runScope)
				}
				report, err = runAnalyze(runScope)
				if err != nil {
					break
				}
				if !shouldEscalateAnalyzeReport(report) || i == len(autoScopes)-1 {
					break
				}
				if verbose {
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] auto escalation requested\n", time.Now().Format("15:04:05"))
				}
			}
		} else {
			report, err = runAnalyze(scope)
		}
	}
	if err != nil {
		// If analyzer returned a partial report with diagnostics, persist diag output first.
		if report != nil && strings.TrimSpace(diagOut) != "" {
			writeDiagnostics(diagOut, report.Diagnostics)
		}
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if strings.TrimSpace(diagOut) != "" {
		writeDiagnostics(diagOut, report.Diagnostics)
	}
	if strings.TrimSpace(graphOut) != "" {
		writeDependencyGraph(graphOut, report.DependencyGraph)
	}
	if lint {
		violations := filterLintDiagnostics(report.Diagnostics, lintSeverity)
		if len(violations) > 0 {
			printDiagnostics(violations, diagGrouped)
			fmt.Fprintf(os.Stderr, "lint failed: %d diagnostic(s) at severity >= %s\n", len(violations), normalizeLintSeverity(lintSeverity))
			os.Exit(1)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] lint ok (severity>=%s)\n", time.Now().Format("15:04:05"), normalizeLintSeverity(lintSeverity))
		}
		return
	}
	if cacheEnabled && cachePath != "" && err == nil {
		if writeErr := writeCachedReport(cachePath, report); writeErr != nil && verbose {
			fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] cache write failed: %v\n", time.Now().Format("15:04:05"), writeErr)
		}
	}
	if diagOnly {
		printDiagnostics(report.Diagnostics, diagGrouped)
		if verbose {
			fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] done in %s\n", time.Now().Format("15:04:05"), time.Since(startedAt).Round(time.Millisecond))
		}
		return
	}

	payload, err := buildPayload(report, emitHook, hookPackage, hookName, exactOnly, toolVersion)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if err := writePayload(out, payload); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] done in %s\n", time.Now().Format("15:04:05"), time.Since(startedAt).Round(time.Millisecond))
	}
}

func newAnalyzeSession(dir string, patterns []string, tags string, strictDI bool, explicitOnly bool, explicitScope string, scope string, toolVersion string, verbose bool, cacheEnabled bool, cacheRoot string, disableCommentDetection bool, disableDirectiveDetection bool) (*analyzer.Session, error) {
	opts := analyzer.Options{
		Dir:           dir,
		Patterns:      patterns,
		Tags:          tags,
		StrictDI:      strictDI,
		DisableCommentDetection: disableCommentDetection,
		DisableDirectiveDetection: disableDirectiveDetection,
		ExplicitOnly:  explicitOnly,
		ExplicitScope: explicitScopeForScope(explicitOnly, explicitScope, scope),
		IndexScope:    scope,
		LoadDeps:      shouldLoadDepsForScope(scope, explicitOnly),
		ToolVersion:   toolVersion,
		Progress: func(message string) {
			if !verbose {
				return
			}
			fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] %s\n", time.Now().Format("15:04:05"), strings.TrimSpace(message))
		},
	}
	if cacheEnabled {
		opts.PackageCacheLoad = func(pkgPath, fingerprint string) (*analyzer.PackageCacheEntry, bool) {
			entry, loadErr := readCachedPackageEntry(cacheRoot, pkgPath, fingerprint)
			return entry, loadErr == nil && entry != nil
		}
		opts.PackageCacheStore = func(pkgPath, fingerprint string, entry analyzer.PackageCacheEntry) {
			if writeErr := writeCachedPackageEntry(cacheRoot, pkgPath, fingerprint, entry); writeErr != nil && verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] package cache write failed (%s): %v\n", time.Now().Format("15:04:05"), strings.TrimSpace(pkgPath), writeErr)
			}
		}
	}
	return analyzer.NewSession(opts)
}

func buildPayload(report *analyzer.Report, emitHook bool, hookPackage, hookFunc string, exactOnly bool, toolVersion string) ([]byte, error) {
	if emitHook {
		src, err := analyzer.GenerateHookSource(report, analyzer.GenerateOptions{
			PackageName: hookPackage,
			FuncName:    hookFunc,
			ExactOnly:   exactOnly,
			ToolVersion: toolVersion,
		})
		if err != nil {
			return nil, err
		}
		return []byte(src), nil
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func writePayload(out string, payload []byte) error {
	if strings.TrimSpace(out) == "" {
		_, err := os.Stdout.Write(payload)
		return err
	}
	return os.WriteFile(out, payload, 0o644)
}

func writeDiagnostics(path string, diagnostics []analyzer.AnalyzerDiagnostic) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	b, err := json.MarshalIndent(diagnostics, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func printDiagnostics(diagnostics []analyzer.AnalyzerDiagnostic, grouped bool) {
	if len(diagnostics) == 0 {
		fmt.Println("no diagnostics")
		return
	}
	if grouped {
		printDiagnosticsGrouped(diagnostics)
		return
	}
	for _, d := range diagnostics {
		printDiagnostic(d)
	}
}

func printDiagnosticsGrouped(diagnostics []analyzer.AnalyzerDiagnostic) {
	type bucket struct {
		label string
		items []analyzer.AnalyzerDiagnostic
	}
	byLabel := map[string][]analyzer.AnalyzerDiagnostic{}
	for _, d := range diagnostics {
		label := primaryDiagnosticLabel(d)
		byLabel[label] = append(byLabel[label], d)
	}
	labels := make([]string, 0, len(byLabel))
	for label := range byLabel {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for idx, label := range labels {
		fmt.Printf("%s\n", label)
		items := byLabel[label]
		sort.Slice(items, func(i, j int) bool {
			if items[i].File == items[j].File {
				if items[i].Line == items[j].Line {
					return items[i].Column < items[j].Column
				}
				return items[i].Line < items[j].Line
			}
			if items[i].Code == items[j].Code {
				return items[i].Message < items[j].Message
			}
			return items[i].Code < items[j].Code
		})
		for _, d := range items {
			printDiagnostic(d)
		}
		if idx != len(labels)-1 {
			fmt.Println()
		}
	}
}

func primaryDiagnosticLabel(d analyzer.AnalyzerDiagnostic) string {
	if len(d.Routes) > 0 {
		routes := append([]string(nil), d.Routes...)
		sort.Strings(routes)
		return "Route: " + strings.Join(routes, ", ")
	}
	if strings.TrimSpace(d.Package) != "" {
		return "Package: " + strings.TrimSpace(d.Package)
	}
	if strings.TrimSpace(d.File) != "" {
		return "File: " + strings.TrimSpace(d.File)
	}
	return "General"
}

func printDiagnostic(d analyzer.AnalyzerDiagnostic) {
	severity := strings.TrimSpace(d.Severity)
	if severity == "" {
		severity = "warning"
	}
	code := strings.TrimSpace(d.Code)
	msg := strings.TrimSpace(d.Message)
	if code != "" {
		fmt.Printf("[%s] %s: %s\n", severity, code, msg)
	} else {
		fmt.Printf("[%s] %s\n", severity, msg)
	}
	if d.File != "" {
		fmt.Printf("  at %s:%d:%d\n", d.File, d.Line, d.Column)
	}
	if d.LineText != "" {
		fmt.Printf("  %s\n", d.LineText)
	}
	if d.Caret != "" {
		fmt.Printf("  %s\n", d.Caret)
	}
}

func computeCacheKey(dir string, patterns []string, tags string, strictDI bool, scope string, toolVersion string, disableCommentDetection bool, disableDirectiveDetection bool) (string, error) {
	const schemaVersion = "autoswag-cache-v1"
	absDir, err := filepath.Abs(strings.TrimSpace(dir))
	if err != nil {
		return "", err
	}
	fingerprint, err := workspaceFingerprint(absDir)
	if err != nil {
		return "", err
	}
	payload := strings.Join([]string{
		schemaVersion,
		"dir=" + absDir,
		"patterns=" + strings.Join(patterns, ","),
		"tags=" + strings.TrimSpace(tags),
		fmt.Sprintf("strict_di=%t", strictDI),
		fmt.Sprintf("disable_comment_detection=%t", disableCommentDetection),
		fmt.Sprintf("disable_directive_detection=%t", disableDirectiveDetection),
		"scope=" + strings.TrimSpace(scope),
		"tool_version=" + strings.TrimSpace(toolVersion),
		"workspace=" + fingerprint,
	}, "\n")
	sum := sha256.Sum256([]byte(payload))
	return fmt.Sprintf("%x", sum[:]), nil
}

func detectToolVersion() string {
	const fallback = "autoswag-analyze-dev"
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil {
		return fallback
	}
	mainPath := strings.TrimSpace(info.Main.Path)
	mainVersion := strings.TrimSpace(info.Main.Version)
	if mainPath == "" {
		mainPath = fallback
	}
	if mainVersion == "" || mainVersion == "(devel)" {
		mainVersion = "devel"
	}
	vcsRevision := ""
	vcsModified := ""
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			vcsRevision = strings.TrimSpace(s.Value)
		case "vcs.modified":
			vcsModified = strings.TrimSpace(s.Value)
		}
	}
	parts := []string{mainPath + "@" + mainVersion}
	if vcsRevision != "" {
		parts = append(parts, "rev="+vcsRevision)
	}
	if vcsModified != "" {
		parts = append(parts, "modified="+vcsModified)
	}
	return strings.Join(parts, "|")
}

func defaultCacheDir() string {
	base := strings.TrimSpace(os.TempDir())
	if base == "" {
		return ".autoswag-cache"
	}
	return filepath.Join(base, "autoswag-cache")
}

func workspaceFingerprint(absDir string) (string, error) {
	entries := []string{}
	err := filepath.WalkDir(absDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == ".autoswag-cache" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") && name != "go.mod" && name != "go.sum" {
			return nil
		}
		if shouldIgnoreAutoswagGeneratedFile(path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}
		entries = append(entries, fmt.Sprintf("%s|%d|%d", filepath.ToSlash(rel), info.Size(), info.ModTime().UnixNano()))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
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

func readCachedReport(path string) (*analyzer.Report, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out analyzer.Report
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func writeCachedReport(path string, report *analyzer.Report) error {
	if report == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(report)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func readCachedPackageEntry(cacheRoot, pkgPath, fingerprint string) (*analyzer.PackageCacheEntry, error) {
	path := packageCacheEntryPath(cacheRoot, pkgPath, fingerprint)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out analyzer.PackageCacheEntry
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func writeCachedPackageEntry(cacheRoot, pkgPath, fingerprint string, entry analyzer.PackageCacheEntry) error {
	path := packageCacheEntryPath(cacheRoot, pkgPath, fingerprint)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func packageCacheEntryPath(cacheRoot, pkgPath, fingerprint string) string {
	cacheRoot = strings.TrimSpace(cacheRoot)
	keySrc := strings.TrimSpace(pkgPath) + "|" + strings.TrimSpace(fingerprint)
	sum := sha256.Sum256([]byte(keySrc))
	name := fmt.Sprintf("%x.json", sum[:])
	return filepath.Join(cacheRoot, "pkgs", name)
}

func shouldEscalateAnalyzeReport(report *analyzer.Report) bool {
	if report == nil {
		return false
	}
	for _, d := range report.Diagnostics {
		switch strings.TrimSpace(d.Code) {
		case "di_unresolved_interface_dispatch", "di_ambiguous_interface_dispatch":
			return true
		}
	}
	for _, p := range report.Packages {
		for _, h := range p.Handlers {
			for _, r := range h.Responses {
				conf := strings.TrimSpace(r.Confidence)
				if conf == "inferred" || conf == "heuristic" {
					for _, step := range r.Trace {
						if strings.HasPrefix(strings.TrimSpace(step), "call:") {
							return true
						}
					}
					if strings.TrimSpace(r.Type) == "any" {
						return true
					}
				}
			}
		}
	}
	return false
}

func normalizeLintSeverity(in string) string {
	switch strings.ToLower(strings.TrimSpace(in)) {
	case "error":
		return "error"
	default:
		return "warning"
	}
}

func filterLintDiagnostics(diags []analyzer.AnalyzerDiagnostic, threshold string) []analyzer.AnalyzerDiagnostic {
	threshold = normalizeLintSeverity(threshold)
	out := make([]analyzer.AnalyzerDiagnostic, 0, len(diags))
	for _, d := range diags {
		sev := strings.ToLower(strings.TrimSpace(d.Severity))
		if threshold == "error" {
			if sev == "error" {
				out = append(out, d)
			}
			continue
		}
		if sev == "error" || sev == "warning" || sev == "" {
			out = append(out, d)
		}
	}
	return out
}

func autoAnalyzeScopes(strictDI bool, explicitOnly bool, explicitScope string) []string {
	if explicitOnly {
		switch strings.ToLower(strings.TrimSpace(explicitScope)) {
		case "auto":
			out := []string{"roots", "imports", "workspace"}
			if strictDI {
				out = append(out, "all")
			}
			return out
		case "roots", "imports", "workspace", "all":
			return []string{strings.ToLower(strings.TrimSpace(explicitScope))}
		default:
			return []string{"imports"}
		}
	}
	out := []string{"roots", "workspace", "referenced"}
	if strictDI {
		out = append(out, "all")
	}
	return out
}

func explicitScopeForScope(explicitOnly bool, explicitScope string, scope string) string {
	if !explicitOnly {
		return ""
	}
	explicitScope = strings.ToLower(strings.TrimSpace(explicitScope))
	if explicitScope == "auto" {
		switch strings.ToLower(strings.TrimSpace(scope)) {
		case "roots":
			return "roots"
		case "workspace", "referenced":
			return "workspace"
		case "all":
			return "all"
		default:
			return "imports"
		}
	}
	return explicitScope
}

func shouldLoadDepsForScope(scope string, explicitOnly bool) bool {
	if explicitOnly {
		return false
	}
	return strings.TrimSpace(strings.ToLower(scope)) == "all"
}

func heapAllocMB() uint64 {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc / (1024 * 1024)
}

func watchRecycleHeapMB() uint64 {
	const def uint64 = 768
	raw := strings.TrimSpace(os.Getenv("AUTOSWAG_WATCH_RECYCLE_HEAP_MB"))
	if raw == "" {
		return def
	}
	n, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || n == 0 {
		return def
	}
	return n
}

func pruneCacheDir(cacheRoot string) error {
	cacheRoot = strings.TrimSpace(cacheRoot)
	if cacheRoot == "" {
		return fmt.Errorf("cache-prune: cache directory is empty")
	}
	if err := os.RemoveAll(cacheRoot); err != nil {
		return fmt.Errorf("cache-prune: remove %s: %w", cacheRoot, err)
	}
	if err := os.MkdirAll(cacheRoot, 0o755); err != nil {
		return fmt.Errorf("cache-prune: recreate %s: %w", cacheRoot, err)
	}
	return nil
}

func writeDependencyGraph(path string, graph *analyzer.DependencyGraph) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	b, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func runWatchMode(cacheRoot, dir string, patterns []string, out string, interval time.Duration, verbose bool, onChange func(changed []string) error) error {
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	root, err := filepath.Abs(strings.TrimSpace(dir))
	if err != nil {
		return fmt.Errorf("watch: resolve dir: %w", err)
	}
	watchRoots, err := deriveWatchRoots(root, patterns)
	if err != nil {
		return err
	}
	excludes := map[string]struct{}{}
	if strings.TrimSpace(out) != "" {
		if absOut, err := absFromCwd(out); err == nil {
			excludes[absOut] = struct{}{}
		}
	}
	if strings.TrimSpace(cacheRoot) != "" {
		if absCache, err := filepath.Abs(strings.TrimSpace(cacheRoot)); err == nil {
			excludes[absCache] = struct{}{}
		}
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watch: create watcher: %w", err)
	}
	defer watcher.Close()
	watchedDirs := map[string]struct{}{}
	for _, wr := range watchRoots {
		if err := addWatchDirsRecursively(watcher, wr, excludes, watchedDirs); err != nil {
			return err
		}
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] watch mode enabled (debounce=%s, roots=%s)\n", time.Now().Format("15:04:05"), interval, strings.Join(pathsToSlash(watchRoots), ","))
	}
	var debounce *time.Timer
	trigger := make(chan struct{}, 1)
	pending := map[string]struct{}{}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	fire := func() {
		select {
		case trigger <- struct{}{}:
		default:
		}
	}
	resetDebounce := func() {
		if debounce != nil {
			debounce.Stop()
		}
		debounce = time.AfterFunc(interval, fire)
	}

	for {
		select {
		case <-sigCh:
			if verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] watch stopped\n", time.Now().Format("15:04:05"))
			}
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if strings.TrimSpace(event.Name) == "" {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}
			absEvent, err := filepath.Abs(event.Name)
			if err != nil {
				continue
			}
			if !isUnderAnyRoot(absEvent, watchRoots) {
				continue
			}
			if shouldExcludePath(absEvent, excludes) {
				continue
			}
			if event.Has(fsnotify.Create) {
				if info, statErr := os.Stat(absEvent); statErr == nil && info.IsDir() {
					_ = addWatchDirsRecursively(watcher, absEvent, excludes, watchedDirs)
					continue
				}
			}
			if !isWatchedSourceFile(absEvent) {
				continue
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] change detected: %s\n", time.Now().Format("15:04:05"), filepath.ToSlash(absEvent))
			}
			pending[absEvent] = struct{}{}
			resetDebounce()
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			if verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] watch error: %v\n", time.Now().Format("15:04:05"), err)
			}
		case <-trigger:
			if verbose {
				fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] rerunning\n", time.Now().Format("15:04:05"))
			}
			changed := make([]string, 0, len(pending))
			for p := range pending {
				changed = append(changed, p)
			}
			sort.Strings(changed)
			pending = map[string]struct{}{}
			if onChange != nil {
				if err := onChange(changed); err != nil && verbose {
					fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] watch run failed: %v\n", time.Now().Format("15:04:05"), err)
				}
			}
		}
	}
}

func absFromCwd(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(cwd, path)), nil
}

func addWatchDirsRecursively(watcher *fsnotify.Watcher, root string, excludes map[string]struct{}, watched map[string]struct{}) error {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return nil
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !d.IsDir() {
			return nil
		}
		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		if shouldExcludePath(absPath, excludes) || shouldSkipWatchDir(d.Name()) {
			return filepath.SkipDir
		}
		if _, ok := watched[absPath]; ok {
			return nil
		}
		if err := watcher.Add(absPath); err != nil {
			return nil
		}
		watched[absPath] = struct{}{}
		return nil
	})
}

func shouldExcludePath(path string, excludes map[string]struct{}) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	for ex := range excludes {
		ex = filepath.Clean(strings.TrimSpace(ex))
		if ex == "" {
			continue
		}
		if path == ex || strings.HasPrefix(path, ex+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func shouldSkipWatchDir(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", "node_modules", "vendor":
		return true
	default:
		return false
	}
}

func isWatchedSourceFile(path string) bool {
	name := strings.TrimSpace(filepath.Base(path))
	if name == "go.mod" || name == "go.sum" {
		return true
	}
	return strings.HasSuffix(name, ".go")
}

func deriveWatchRoots(root string, patterns []string) ([]string, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "" {
		return nil, fmt.Errorf("watch: empty root")
	}
	if len(patterns) == 0 {
		return []string{root}, nil
	}
	out := []string{}
	seen := map[string]struct{}{}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" || p == "." {
			p = root
		} else {
			if idx := strings.Index(p, "..."); idx >= 0 {
				p = p[:idx]
			}
			p = strings.TrimSuffix(p, "/")
			if p == "" {
				p = root
			} else if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		abs = filepath.Clean(abs)
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	if len(out) == 0 {
		out = append(out, root)
	}
	sort.Strings(out)
	return out, nil
}

func isUnderAnyRoot(path string, roots []string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	for _, r := range roots {
		r = filepath.Clean(strings.TrimSpace(r))
		if r == "" {
			continue
		}
		if path == r || strings.HasPrefix(path, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func pathsToSlash(in []string) []string {
	out := make([]string, 0, len(in))
	for _, p := range in {
		out = append(out, filepath.ToSlash(p))
	}
	return out
}
