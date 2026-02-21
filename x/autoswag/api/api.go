package api

import (
	"encoding/json"

	"github.com/bronystylecrazy/ultrastructure/x/autoswag/analyzer"
)

type Request struct {
	AnalyzerOptions analyzer.Options
	EmitHook        bool
	HookPackage     string
	HookName        string
	ExactOnly       bool
	ToolVersion     string
}

type Result struct {
	Report  *analyzer.Report
	Payload []byte
}

func Run(req Request) (*Result, error) {
	report, err := analyzer.Analyze(req.AnalyzerOptions)
	if err != nil {
		return nil, err
	}
	payload, err := BuildPayload(report, PayloadOptions{
		EmitHook:    req.EmitHook,
		HookPackage: req.HookPackage,
		HookName:    req.HookName,
		ExactOnly:   req.ExactOnly,
		ToolVersion: req.ToolVersion,
	})
	if err != nil {
		return nil, err
	}
	return &Result{
		Report:  report,
		Payload: payload,
	}, nil
}

type PayloadOptions struct {
	EmitHook    bool
	HookPackage string
	HookName    string
	ExactOnly   bool
	ToolVersion string
}

func BuildPayload(report *analyzer.Report, opts PayloadOptions) ([]byte, error) {
	if opts.EmitHook {
		src, err := analyzer.GenerateHookSource(report, analyzer.GenerateOptions{
			PackageName: opts.HookPackage,
			FuncName:    opts.HookName,
			ExactOnly:   opts.ExactOnly,
			ToolVersion: opts.ToolVersion,
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
