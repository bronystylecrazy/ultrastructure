package di

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Diagnostics installs a minimal error hook.
func Diagnostics() Node {
	if os.Getenv("RUN_DEBUG") == "" {
		return diagnosticsNode{simple: false}
	}
	return nil
}

type diagnosticsNode struct {
	simple bool
}

func (n diagnosticsNode) Build() (fx.Option, error) {
	return fx.WithLogger(func(in struct {
		fx.In
		Logger *zap.Logger `optional:"true"`
	}) fxevent.Logger {
		logger := in.Logger
		if logger == nil {
			logger = newDiagnosticsFallbackLogger()
		}
		logger = logger.WithOptions(zap.AddStacktrace(zapcore.FatalLevel))
		return &diagnosticsLogger{simple: n.simple, logger: logger}
	}), nil
}

func newDiagnosticsFallbackLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	logger, err := cfg.Build()
	if err != nil {
		return zap.NewNop()
	}
	return logger
}

type diagnosticsLogger struct {
	simple bool
	logger *zap.Logger
	once   sync.Once
}

func (l *diagnosticsLogger) LogEvent(event fxevent.Event) {
	var err error
	// Extract error from any Fx event type that carries one.
	switch e := event.(type) {
	case *fxevent.Started:
		err = e.Err
	case *fxevent.Stopped:
		err = e.Err
	case *fxevent.Invoked:
		err = e.Err
	case *fxevent.Provided:
		err = e.Err
	case *fxevent.Decorated:
		err = e.Err
	case *fxevent.Supplied:
		err = e.Err
	case *fxevent.Run:
		err = e.Err
	case *fxevent.OnStartExecuted:
		err = e.Err
	case *fxevent.OnStopExecuted:
		err = e.Err
	case *fxevent.RollingBack:
		err = e.StartErr
	case *fxevent.RolledBack:
		err = e.Err
	}
	if err == nil {
		return
	}
	l.once.Do(func() {
		l.logger.Error("fx error\n" + formatDiagnostics(err, l.simple))
	})
}

type errorWithGraph interface {
	Graph() fx.DotGraph
}

type diagLocation struct {
	file string
	line int
	col  int
}

type diagSpan struct {
	loc   diagLocation
	label string
	score int
}

type diagLabelKind int

const (
	diagLabelGeneric diagLabelKind = iota
	diagLabelWiring
	diagLabelSignature
)

var goFileLineRx = regexp.MustCompile(`([^\s:]+\.go):(\d+)`)
var goFileLineColRx = regexp.MustCompile(`([^\s:]+\.go):(\d+):(\d+)`)

func formatDiagnostics(err error, simple bool) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	spans := parseSpans(msg)
	missingTypes := extractMissingTypes(msg)
	panicMsg := extractPanicMessage(msg)
	limit := 2
	if simple {
		limit = 1
	}
	spans = selectBestSpans(spans, limit)
	if len(spans) == 0 {
		return fmt.Sprintf("error: %s\n", msg)
	}

	lines := strings.Split(msg, "\n")
	header := lines[0]
	var rest string
	if len(lines) > 1 {
		rest = filterDiagnosticRest(lines[1:])
	}

	var b strings.Builder
	b.WriteString("error: ")
	b.WriteString(header)
	b.WriteString("\n")

	for idx, span := range spans {
		loc := resolveLocation(span.loc)
		label := strings.TrimSpace(span.label)
		kind, section := classifyDiagLabel(label)
		if label == "" && len(missingTypes) > 0 {
			label = "missing type: " + missingTypes[0]
		}
		block, caretLines, ok, matched := readSnippetBlock(loc.file, loc.line, 3, loc.col, missingTypes, panicMsg, simple)
		if !ok {
			if idx > 0 {
				b.WriteString("\n")
			}
			b.WriteString("  --> ")
			b.WriteString(loc.file)
			b.WriteString(":")
			b.WriteString(strconv.Itoa(loc.line))
			b.WriteString("\n")
			continue
		}
		if len(missingTypes) > 0 && !matched {
			continue
		}

		if idx > 0 {
			b.WriteString("\n")
		}

		lineNo := strconv.Itoa(loc.line)
		if label == "" {
			label = "here"
		}
		if simple {
			label = ""
		}
		if len(label) > 120 {
			label = label[:117] + "..."
		}

		b.WriteString(" --> ")
		b.WriteString(loc.file)
		b.WriteString(":")
		b.WriteString(lineNo)
		if section != "" && !simple {
			b.WriteString(" [")
			b.WriteString(section)
			b.WriteString("]")
		}
		b.WriteString("\n")
		width := lineNumberWidth(block.lines)
		var content []string
		for _, line := range block.lines {
			num := strconv.Itoa(line.num)
			content = append(content, fmt.Sprintf("%s%s │ %s", strings.Repeat(" ", width-len(num)), num, line.text))
			if line.num == loc.line {
				if kind != diagLabelGeneric {
					label = ""
				}
				if len(caretLines) == 0 && label != "" {
					caretLines = []string{"^ " + label}
				} else if !simple && label != "here" {
					caretLines = append(caretLines, "^ "+label)
				}
				// no spacer line for simple output
				for i, caret := range caretLines {
					if simple && i == 0 {
						content = append(content, fmt.Sprintf("%s │ ┌%s", strings.Repeat(" ", width), caret))
						continue
					}
					prefix := "|"
					if simple {
						prefix = "│"
					}
					content = append(content, fmt.Sprintf("%s %s %s", strings.Repeat(" ", width), prefix, caret))
				}
				if simple && len(missingTypes) > 0 {
					content = append(content, fmt.Sprintf("%s │ └─ missing: %s", strings.Repeat(" ", width), strings.Join(missingTypes, ", ")))
				} else if simple && panicMsg != "" {
					content = append(content, fmt.Sprintf("%s │ └─ panic: %s", strings.Repeat(" ", width), panicMsg))
				}
			}
		}
		b.WriteString(" │")
		b.WriteString("\n")
		for _, line := range content {
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if rest != "" && !simple {
		writeDiagnosticRest(&b, rest)
	}

	if !simple {
		var ewq errorWithGraph
		if errors.As(err, &ewq) {
			b.WriteString("note: a dependency graph is available via fx.DotGraph\n")
		}
	}

	return b.String()
}

func parseSpans(msg string) []diagSpan {
	lines := strings.Split(msg, "\n")
	var spans []diagSpan
	for _, line := range lines {
		spans = append(spans, parseLineSpans(line)...)
	}
	return dedupeSpans(spans)
}

func parseLineSpans(line string) []diagSpan {
	var spans []diagSpan
	matches := goFileLineColRx.FindAllStringSubmatchIndex(line, -1)
	if len(matches) == 0 {
		matches = goFileLineRx.FindAllStringSubmatchIndex(line, -1)
	}
	if len(matches) == 0 {
		return nil
	}

	for _, match := range matches {
		file := line[match[2]:match[3]]
		lineNo, err := strconv.Atoi(line[match[4]:match[5]])
		if err != nil {
			continue
		}
		col := 0
		if len(match) >= 8 && match[6] != -1 {
			col, _ = strconv.Atoi(line[match[6]:match[7]])
		}
		file = cleanFileToken(file)
		label := strings.TrimSpace(strings.Replace(line, line[match[0]:match[1]], "", 1))
		label = strings.TrimLeft(label, ": ")
		label = strings.Trim(label, "()-")
		spans = append(spans, diagSpan{
			loc:   diagLocation{file: file, line: lineNo, col: col},
			label: label,
			score: scoreLine(line, label),
		})
	}
	return spans
}

func cleanFileToken(file string) string {
	file = strings.TrimSpace(file)
	file = strings.TrimLeft(file, "(")
	file = strings.TrimRight(file, "):")
	return file
}

func filterDiagnosticRest(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	var kept []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if goFileLineColRx.MatchString(trimmed) || goFileLineRx.MatchString(trimmed) {
			continue
		}
		kept = append(kept, trimmed)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

func writeDiagnosticRest(b *strings.Builder, rest string) {
	lines := strings.Split(rest, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "hint:") {
			b.WriteString(trimmed)
			b.WriteString("\n")
			continue
		}
		b.WriteString("note: ")
		b.WriteString(trimmed)
		b.WriteString("\n")
	}
}

func classifyDiagLabel(label string) (diagLabelKind, string) {
	lower := strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(lower, "di wiring"):
		return diagLabelWiring, "WIRING"
	case strings.Contains(lower, "signature"):
		return diagLabelSignature, "SIGNATURE"
	default:
		return diagLabelGeneric, ""
	}
}

func scoreLine(line string, label string) int {
	score := 0
	lineLower := strings.ToLower(line)
	if strings.Contains(lineLower, "missing dependencies") {
		score += 4
	}
	if strings.Contains(lineLower, "missing type") {
		score += 4
	}
	if strings.Contains(lineLower, "panic:") {
		score += 6
	}
	if strings.HasSuffix(strings.TrimSpace(line), ":") {
		score += 3
	}
	if strings.Contains(lineLower, "called from") {
		score -= 4
	}
	if strings.Contains(lineLower, "runtime.") {
		score -= 3
	}
	if label != "" {
		score++
	}
	return score
}

func extractMissingTypes(msg string) []string {
	var out []string
	lines := strings.Split(msg, "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.Contains(line, "missing types:") {
			out = append(out, splitMissingTypes(line)...)
			continue
		}
		if strings.HasPrefix(line, "missing type:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "missing type:"))
			if val != "" {
				out = append(out, val)
			}
			continue
		}
		if strings.HasPrefix(line, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "- "))
			if val != "" {
				out = append(out, val)
			}
		}
	}
	if len(out) > 0 {
		return uniqueStrings(out)
	}
	if idx := strings.Index(msg, "missing type:"); idx >= 0 {
		rest := msg[idx+len("missing type:"):]
		rest = strings.TrimSpace(rest)
		if rest != "" {
			fields := strings.Fields(rest)
			if len(fields) > 0 {
				return []string{fields[0]}
			}
		}
	}
	if idx := strings.Index(msg, "missing types:"); idx >= 0 {
		rest := msg[idx+len("missing types:"):]
		return uniqueStrings(splitMissingTypes(rest))
	}
	return nil
}

func splitMissingTypes(line string) []string {
	if idx := strings.Index(line, "missing types:"); idx >= 0 {
		line = line[idx+len("missing types:"):]
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	parts := strings.FieldsFunc(line, func(r rune) bool {
		return r == ';' || r == ',' || r == '│'
	})
	var out []string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		part = strings.TrimSuffix(part, ".")
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func extractPanicMessage(msg string) string {
	idx := strings.Index(msg, "panic:")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(msg[idx+len("panic:"):])
	if rest == "" {
		return ""
	}
	if in := strings.Index(rest, " in func:"); in >= 0 {
		rest = strings.TrimSpace(rest[:in])
	}
	return rest
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func selectBestSpans(spans []diagSpan, limit int) []diagSpan {
	if len(spans) == 0 {
		return spans
	}
	sort.SliceStable(spans, func(i, j int) bool {
		if spans[i].score == spans[j].score {
			return spans[i].loc.line < spans[j].loc.line
		}
		return spans[i].score > spans[j].score
	})
	if limit > 0 && len(spans) > limit {
		spans = spans[:limit]
	}
	return spans
}

func dedupeSpans(spans []diagSpan) []diagSpan {
	if len(spans) == 0 {
		return spans
	}
	seen := make(map[string]struct{}, len(spans))
	out := make([]diagSpan, 0, len(spans))
	for _, span := range spans {
		key := fmt.Sprintf("%s:%d:%d:%s", span.loc.file, span.loc.line, span.loc.col, span.label)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, span)
	}
	return out
}

func resolveLocation(loc diagLocation) diagLocation {
	cwd, err := os.Getwd()
	if err == nil {
		if filepath.IsAbs(loc.file) && strings.HasPrefix(loc.file, cwd) {
			return loc
		}
		abs := filepath.Join(cwd, loc.file)
		if _, err := os.Stat(abs); err == nil {
			return diagLocation{file: abs, line: loc.line}
		}
	}
	return loc
}

type snippetLine struct {
	num  int
	text string
}

type snippetBlock struct {
	lines []snippetLine
}

func lineNumberWidth(lines []snippetLine) int {
	max := 0
	for _, line := range lines {
		if line.num > max {
			max = line.num
		}
	}
	if max == 0 {
		return 1
	}
	return len(strconv.Itoa(max))
}

func readSnippetBlock(path string, lineNo int, context int, col int, tokens []string, panicMsg string, simple bool) (snippetBlock, []string, bool, bool) {
	if lineNo <= 0 {
		return snippetBlock{}, nil, false, false
	}

	file, err := os.Open(path)
	if err != nil {
		return snippetBlock{}, nil, false, false
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	current := 1
	start := lineNo - context
	end := lineNo + context
	if start < 1 {
		start = 1
	}
	block := snippetBlock{}
	for scanner.Scan() {
		if current < start {
			current++
			continue
		}
		if current > end {
			break
		}
		if current == lineNo {
			line := expandTabs(scanner.Text())
			caret, matched := caretForLine(line, col, tokens, panicMsg, simple)
			block.lines = append(block.lines, snippetLine{num: current, text: line})
			return block, caret, true, matched
		}
		block.lines = append(block.lines, snippetLine{num: current, text: expandTabs(scanner.Text())})
		current++
	}

	if len(block.lines) == 0 {
		return snippetBlock{}, nil, false, false
	}
	return block, nil, true, false
}

func expandTabs(line string) string {
	return strings.ReplaceAll(line, "\t", "    ")
}

func caretForLine(line string, col int, tokens []string, panicMsg string, simple bool) ([]string, bool) {
	trimmed := strings.TrimRight(line, " \t")
	if trimmed == "" {
		return []string{"^"}, false
	}

	if col == 0 && len(tokens) > 0 {
		if carets, ok := caretLinesForTokens(trimmed, tokens, simple); ok {
			return carets, true
		}
	}
	if col == 0 && len(tokens) == 0 && panicMsg != "" {
		if caret, ok := caretForPanic(trimmed); ok {
			return []string{caret}, true
		}
	}

	if col > 0 && col <= len(trimmed) {
		return []string{strings.Repeat(" ", col-1) + "^"}, false
	}

	start := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] != ' ' {
			start = i
			break
		}
	}

	length := len(trimmed) - start
	if length <= 0 {
		length = 1
	}
	if length > 120 {
		length = 120
	}

	return []string{strings.Repeat(" ", start) + strings.Repeat("^", length)}, false
}

type caretSpan struct {
	start  int
	length int
	label  string
}

func caretLinesForTokens(line string, tokens []string, simple bool) ([]string, bool) {
	type span struct {
		start  int
		length int
		label  string
	}
	var spans []caretSpan
	for _, token := range tokens {
		if token == "" {
			continue
		}
		start, length, ok := spanForToken(line, token)
		if !ok {
			continue
		}
		spans = append(spans, caretSpan{
			start:  start,
			length: length,
			label:  "missing type: " + token,
		})
	}
	if len(spans) == 0 {
		return nil, false
	}
	if simple {
		return []string{singleCaretLine(line, spans)}, true
	}
	lines := make([]string, 0, len(spans))
	for _, sp := range spans {
		caret := strings.Repeat(" ", sp.start) + strings.Repeat("^", sp.length)
		lines = append(lines, caret+" "+sp.label)
	}
	return lines, true
}

func singleCaretLine(line string, spans []caretSpan) string {
	maxEnd := 0
	firstStart := -1
	for _, sp := range spans {
		end := sp.start + sp.length
		if end > maxEnd {
			maxEnd = end
		}
		if firstStart == -1 || sp.start < firstStart {
			firstStart = sp.start
		}
	}
	if maxEnd == 0 {
		return "^"
	}
	buf := make([]rune, maxEnd)
	for i := range buf {
		buf[i] = ' '
	}
	for _, sp := range spans {
		for i := 0; i < sp.length && sp.start+i < len(buf); i++ {
			buf[sp.start+i] = '^'
		}
	}
	if firstStart > 0 {
		for i := 0; i < firstStart; i++ {
			if buf[i] == ' ' {
				buf[i] = '─'
			}
		}
	}
	return string(buf)
}

func caretForPanic(line string) (string, bool) {
	start, length, ok := spanForPanic(line)
	if !ok {
		return "", false
	}
	return strings.Repeat(" ", start) + strings.Repeat("^", length), true
}

func spanForPanic(line string) (int, int, bool) {
	start := strings.Index(line, "panic(")
	if start < 0 {
		return 0, 0, false
	}
	end := strings.Index(line[start:], ")")
	if end >= 0 {
		end = start + end + 1
	} else {
		end = start + len("panic(")
	}
	if end <= start {
		return 0, 0, false
	}
	return start, end - start, true
}

func spanForToken(line string, token string) (int, int, bool) {
	if start, length, ok := spanForParam(line, token); ok {
		return start, length, true
	}
	idx := strings.Index(line, token)
	if idx < 0 {
		return 0, 0, false
	}
	start := idx
	end := idx + len(token)

	// Try to extend left to include parameter name: "<ident> <token>".
	left := strings.TrimRight(line[:idx], " \t")
	if left != "" {
		i := len(left) - 1
		for i >= 0 && ((left[i] >= 'a' && left[i] <= 'z') || (left[i] >= 'A' && left[i] <= 'Z') || (left[i] >= '0' && left[i] <= '9') || left[i] == '_') {
			i--
		}
		identStart := i + 1
		if identStart < len(left) {
			start = identStart
		}
	}

	return start, end - start, true
}

func spanForParam(line string, token string) (int, int, bool) {
	open := strings.Index(line, "func(")
	if open >= 0 {
		open += len("func")
	} else {
		open = strings.Index(line, "(")
	}
	if open == -1 {
		return 0, 0, false
	}
	close := strings.Index(line[open:], ")")
	if close >= 0 {
		close = open + close
	}
	if open == -1 || close == -1 || close <= open {
		return 0, 0, false
	}

	params := line[open+1 : close]
	typeName := strings.TrimPrefix(token, "*")
	if dot := strings.LastIndex(token, "."); dot >= 0 {
		typeName = strings.TrimPrefix(token[dot+1:], "*")
	}

	offset := open + 1
	segments := strings.Split(params, ",")
	for _, seg := range segments {
		raw := seg
		seg = strings.TrimSpace(seg)
		if seg == "" {
			offset += len(raw) + 1
			continue
		}
		if strings.Contains(seg, token) || containsTypeName(seg, typeName) {
			trimLeft := len(raw) - len(strings.TrimLeft(raw, " \t"))
			start := offset + trimLeft
			return start, len(seg), true
		}
		offset += len(raw) + 1
	}

	return 0, 0, false
}

func containsTypeName(seg string, typeName string) bool {
	if typeName == "" {
		return false
	}
	if strings.Contains(seg, "."+typeName) {
		return true
	}
	parts := strings.Fields(seg)
	if len(parts) == 0 {
		return false
	}
	last := parts[len(parts)-1]
	last = strings.TrimPrefix(last, "*")
	return last == typeName
}
