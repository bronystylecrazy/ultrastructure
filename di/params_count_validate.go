package di

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
)

func validateParamCountForFunction(function any, cfg paramConfig, sourceFile string, sourceLine int, mismatchFormat string, signatureLabel string) error {
	if !cfg.paramsSet {
		return nil
	}
	if function == nil {
		return nil
	}
	fn := reflect.TypeOf(function)
	if fn == nil || fn.Kind() != reflect.Func {
		return nil
	}
	return validateParamCountForFunctionExpected(function, cfg, sourceFile, sourceLine, mismatchFormat, signatureLabel, fn.NumIn())
}

func validateParamCountForFunctionExpected(function any, cfg paramConfig, sourceFile string, sourceLine int, mismatchFormat string, signatureLabel string, expected int) error {
	if !cfg.paramsSet {
		return nil
	}
	if function == nil {
		return nil
	}
	fn := reflect.TypeOf(function)
	if fn == nil || fn.Kind() != reflect.Func {
		return nil
	}
	if cfg.paramSlots == expected {
		return nil
	}
	baseMsg := fmt.Sprintf(mismatchFormat, expected, cfg.paramSlots)
	wiringFile := sourceFile
	wiringLine := sourceLine
	if cfg.paramsSourceFile != "" && cfg.paramsSourceLine > 0 {
		wiringFile = cfg.paramsSourceFile
		wiringLine = cfg.paramsSourceLine
	}

	signatureFile := ""
	signatureLine := 0
	if runtimeFn := runtime.FuncForPC(reflect.ValueOf(function).Pointer()); runtimeFn != nil {
		signatureFile, signatureLine = runtimeFn.FileLine(reflect.ValueOf(function).Pointer())
	}

	if wiringFile == "" || wiringLine <= 0 {
		if signatureFile != "" && signatureLine > 0 {
			return fmt.Errorf("%s:%d: %s", signatureFile, signatureLine, baseMsg)
		}
		return errors.New(baseMsg)
	}
	if signatureFile == wiringFile && signatureLine == wiringLine {
		return fmt.Errorf("%s\n%s:%d: di wiring: %s", baseMsg, wiringFile, wiringLine, baseMsg)
	}
	if signatureFile != "" && signatureLine > 0 {
		return fmt.Errorf("%s\n%s:%d: di wiring:\n%s:%d: %s:", baseMsg, wiringFile, wiringLine, signatureFile, signatureLine, signatureLabel)
	}
	return fmt.Errorf("%s\n%s:%d: di wiring:", baseMsg, wiringFile, wiringLine)
}
