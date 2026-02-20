package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer"
)

func main() {
	startedAt := time.Now()

	var dir string
	var pattern string
	var emitHook bool
	var hookPackage string
	var hookFunc string
	var out string
	var tags string
	var exactOnly bool
	var strictDI bool
	var diagOut string
	var graphOut string
	var verbose bool
	var deep bool
	var indexScope string
	var diagOnly bool
	var diagGrouped bool
	var cacheDir string
	var noCache bool
	var cachePrune bool
	flag.StringVar(&dir, "dir", ".", "working directory")
	flag.StringVar(&pattern, "patterns", ".", "comma-separated go package patterns")
	flag.BoolVar(&emitHook, "emit-hook", false, "emit autoswag WithCustomize hook source instead of JSON report")
	flag.StringVar(&hookPackage, "hook-package", "autoswaggen", "package name for generated hook source")
	flag.StringVar(&hookFunc, "hook-func", "AutoDetectedSwaggerHook", "function name for generated hook source")
	flag.StringVar(&out, "out", "", "write output to file path (default: stdout)")
	flag.StringVar(&tags, "tags", "", "build tags for package loading (comma-separated or go -tags expression)")
	flag.BoolVar(&exactOnly, "exact-only", false, "when -emit-hook is enabled, include only exact-confidence detections")
	flag.BoolVar(&strictDI, "strict-di", false, "treat ambiguous DI interface dispatch as an error")
	flag.StringVar(&diagOut, "diag-out", "", "write analyzer diagnostics JSON to file path")
	flag.StringVar(&graphOut, "graph-out", "", "write dependency graph JSON to file path")
	flag.BoolVar(&verbose, "verbose", false, "print analyzer progress logs to stderr")
	flag.BoolVar(&deep, "deep", true, "enable deep indexing (equivalent to --index-scope all when index-scope is not set)")
	flag.StringVar(&indexScope, "index-scope", "", "index scope: workspace|roots|referenced|all (overrides --deep)")
	flag.BoolVar(&diagOnly, "diag-only", false, "print diagnostics only (human-readable), do not emit JSON report/hook payload")
	flag.BoolVar(&diagGrouped, "diag-grouped", true, "group diagnostics by route/package in -diag-only output")
	flag.StringVar(&cacheDir, "cache-dir", defaultCacheDir(), "directory for analysis cache")
	flag.BoolVar(&noCache, "no-cache", false, "disable reading/writing analysis cache")
	flag.BoolVar(&cachePrune, "cache-prune", false, "delete cache directory and exit")
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
			scope = "all"
		} else {
			scope = "workspace"
		}
	}

	patterns := []string{"."}
	if strings.TrimSpace(pattern) != "" {
		patterns = strings.Split(pattern, ",")
		for i := range patterns {
			patterns[i] = strings.TrimSpace(patterns[i])
		}
	}
	toolVersion := detectToolVersion()

	var report *analyzer.Report
	var err error
	cacheKey := ""
	cachePath := ""
	cacheEnabled := !noCache && cacheRoot != ""
	if cacheEnabled {
		cacheKey, err = computeCacheKey(dir, patterns, tags, strictDI, scope, toolVersion)
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
		opts := analyzer.Options{
			Dir:         dir,
			Patterns:    patterns,
			Tags:        tags,
			StrictDI:    strictDI,
			IndexScope:  scope,
			ToolVersion: toolVersion,
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
		report, err = analyzer.Analyze(opts)
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

	var payload []byte
	if emitHook {
		src, err := analyzer.GenerateHookSource(report, analyzer.GenerateOptions{
			PackageName: hookPackage,
			FuncName:    hookFunc,
			ExactOnly:   exactOnly,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		payload = []byte(src)
	} else {
		b, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		payload = append(b, '\n')
	}

	if strings.TrimSpace(out) == "" {
		if _, err := os.Stdout.Write(payload); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] done in %s\n", time.Now().Format("15:04:05"), time.Since(startedAt).Round(time.Millisecond))
		}
		return
	}
	if err := os.WriteFile(out, payload, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	if verbose {
		fmt.Fprintf(os.Stderr, "[autoswag-analyze %s] done in %s\n", time.Now().Format("15:04:05"), time.Since(startedAt).Round(time.Millisecond))
	}
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

func computeCacheKey(dir string, patterns []string, tags string, strictDI bool, scope string, toolVersion string) (string, error) {
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
