package service

import (
	"fmt"
	"strings"

	kservice "github.com/kardianos/service"
	"go.uber.org/zap"
)

type runtimeProgram struct {
	log    *zap.Logger
	sysLog kservice.Logger
}

func (p *runtimeProgram) Start(s kservice.Service) error {
	if p.log != nil {
		p.log.Info("service runtime start requested")
	}
	if p.sysLog != nil {
		_ = p.sysLog.Info("service runtime start requested")
	}
	return nil
}

func (p *runtimeProgram) Stop(s kservice.Service) error {
	if p.log != nil {
		p.log.Info("service runtime stop requested")
	}
	if p.sysLog != nil {
		_ = p.sysLog.Info("service runtime stop requested")
	}
	return nil
}

// RuntimeMode reports whether process is running under a service manager.
func RuntimeMode() (Mode, error) {
	if kservice.Interactive() {
		return ModeCLI, nil
	}
	return ModeDaemon, nil
}

// MaybeRunWindowsService runs the process in service-manager mode when applicable.
// Returns handled=true when the process was started by service manager.
func MaybeRunWindowsService(name string, log *zap.Logger) (bool, error) {
	mode, err := RuntimeMode()
	if err != nil {
		return false, err
	}
	if mode != ModeDaemon {
		return false, nil
	}

	serviceName := strings.TrimSpace(name)
	if serviceName == "" {
		return true, fmt.Errorf("service name is empty")
	}

	program := &runtimeProgram{log: log}
	svc, err := kservice.New(program, &kservice.Config{
		Name:        serviceName,
		DisplayName: serviceName,
		Description: serviceName,
	})
	if err != nil {
		return true, fmt.Errorf("create runtime service %q: %w", serviceName, err)
	}

	sysLog, sysLogErr := svc.SystemLogger(nil)
	if sysLogErr == nil {
		program.sysLog = sysLog
	}

	if log != nil {
		log.Info("service runtime started", zap.String("service", serviceName), zap.String("platform", svc.Platform()))
	}
	if program.sysLog != nil {
		_ = program.sysLog.Infof("service runtime started (platform=%s)", svc.Platform())
	}

	if err := svc.Run(); err != nil {
		return true, fmt.Errorf("run service %q: %w", serviceName, err)
	}
	return true, nil
}
