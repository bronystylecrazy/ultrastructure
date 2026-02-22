package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bronystylecrazy/ultrastructure/di"
	"github.com/bronystylecrazy/ultrastructure/lc"
	"github.com/bronystylecrazy/ultrastructure/meta"
	xservice "github.com/bronystylecrazy/ultrastructure/service"
	"github.com/kardianos/service"
)

func UseServiceController() di.Node {
	return di.Provide(
		NewServiceController,
		di.As[ServiceController](),
		di.AutoGroupIgnoreType[lc.Starter](),
		di.AutoGroupIgnoreType[lc.Stopper](),
	)
}

type serviceProgram struct{}

func (p *serviceProgram) Start(s service.Service) error { return nil }
func (p *serviceProgram) Stop(s service.Service) error  { return nil }

type systemServiceController struct {
	service        service.Service
	name           string
	logDir         string
	windowsLogFile string
}

func NewServiceController() (*systemServiceController, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable path: %w", err)
	}
	name := sanitizeServiceName(meta.Name)
	logDir := defaultUserLogDir()
	_ = os.MkdirAll(logDir, 0o755)

	options := service.KeyValue{
		"LogOutput": true,
	}
	if runtime.GOOS == "darwin" {
		options["UserService"] = true
		options["LogDirectory"] = logDir
		options["RunAtLoad"] = true
	} else if runtime.GOOS == "linux" {
		options["UserService"] = isNixOS()
		options["LogDirectory"] = logDir
	} else if runtime.GOOS != "windows" {
		options["UserService"] = true
		options["LogDirectory"] = logDir
	}

	svc, err := service.New(&serviceProgram{}, &service.Config{
		Name:        name,
		DisplayName: meta.Name,
		Description: meta.Description,
		Executable:  exe,
		Arguments:   []string{},
		Option:      options,
	})
	if err != nil {
		return nil, fmt.Errorf("create service config: %w", err)
	}

	return &systemServiceController{
		service:        svc,
		name:           name,
		logDir:         logDir,
		windowsLogFile: xservice.WindowsServiceLogFile(name),
	}, nil
}

func (c *systemServiceController) Install(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	_, installed, err := c.status()
	if err != nil {
		return err
	}
	if installed {
		return nil
	}

	err = c.service.Install()
	if err == nil {
		return nil
	}

	_, installed, statusErr := c.status()
	if statusErr == nil && installed {
		return nil
	}
	return fmt.Errorf("install service: %w", err)
}

func (c *systemServiceController) Uninstall(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	status, installed, err := c.status()
	if err != nil {
		return err
	}
	if !installed {
		return nil
	}
	if status == service.StatusRunning {
		if err := c.Stop(ctx); err != nil {
			return err
		}
	}

	err = c.service.Uninstall()
	if err == nil {
		return nil
	}

	_, installed, statusErr := c.status()
	if statusErr == nil && !installed {
		return nil
	}
	return fmt.Errorf("uninstall service: %w", err)
}

func (c *systemServiceController) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	status, installed, err := c.status()
	if err != nil {
		return err
	}
	if !installed {
		return fmt.Errorf("start service: %w", service.ErrNotInstalled)
	}
	if status == service.StatusRunning {
		return nil
	}

	err = c.service.Start()
	if err == nil {
		return nil
	}

	status, installed, statusErr := c.status()
	if statusErr == nil && installed && status == service.StatusRunning {
		return nil
	}
	return fmt.Errorf("start service: %w", err)
}

func (c *systemServiceController) Stop(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	status, installed, err := c.status()
	if err != nil {
		return err
	}
	if !installed || status == service.StatusStopped {
		return nil
	}

	err = c.service.Stop()
	if err == nil {
		return nil
	}

	status, installed, statusErr := c.status()
	if statusErr == nil && (!installed || status == service.StatusStopped) {
		return nil
	}
	return fmt.Errorf("stop service: %w", err)
}

func (c *systemServiceController) Restart(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err := c.Stop(ctx); err != nil {
		return err
	}
	return c.Start(ctx)
}

func (c *systemServiceController) status() (service.Status, bool, error) {
	status, err := c.service.Status()
	if err == nil {
		return status, true, nil
	}
	if errors.Is(err, service.ErrNotInstalled) {
		return service.StatusUnknown, false, nil
	}
	return service.StatusUnknown, false, fmt.Errorf("service status: %w", err)
}

func (c *systemServiceController) Status(ctx context.Context, out io.Writer, follow bool) error {
	status, installed, err := c.status()
	if err != nil {
		return err
	}

	switch {
	case !installed:
		_, _ = fmt.Fprintln(out, "status: not-installed")
	case status == service.StatusRunning:
		_, _ = fmt.Fprintln(out, "status: running")
	case status == service.StatusStopped:
		_, _ = fmt.Fprintln(out, "status: stopped")
	default:
		_, _ = fmt.Fprintln(out, "status: unknown")
	}

	if c.windowsLogFile != "" {
		_, _ = fmt.Fprintf(out, "log file: %s\n", c.windowsLogFile)
	}

	if !follow {
		return nil
	}
	if !installed || status != service.StatusRunning {
		_, _ = fmt.Fprintln(out, "log stream disabled: service is not running")
		return nil
	}

	_, _ = fmt.Fprintln(out, "streaming logs (Ctrl+C to stop)...")
	if c.windowsLogFile != "" {
		return followSingleLogFile(ctx, out, c.windowsLogFile)
	}
	return followLogFiles(ctx, out, c.logCandidates())
}

func (c *systemServiceController) logCandidates() []string {
	if c.windowsLogFile != "" {
		return []string{c.windowsLogFile}
	}

	name := c.name
	dir := c.logDir
	return []string{
		filepath.Join(dir, name+".out.log"),
		filepath.Join(dir, name+".err.log"),
		filepath.Join(dir, name+".out"),
		filepath.Join(dir, name+".err"),
		filepath.Join(dir, name+".log"),
	}
}

func defaultUserLogDir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}
	if runtime.GOOS == "darwin" {
		return home
	}
	return filepath.Join(home, ".local", "state")
}

func isNixOS() bool {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return false
	}
	content := strings.ToLower(string(data))
	return strings.Contains(content, "id=nixos") || strings.Contains(content, "id_like=nixos")
}

func followSingleLogFile(ctx context.Context, out io.Writer, path string) error {
	var offset int64

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		f, err := os.Open(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				time.Sleep(300 * time.Millisecond)
				continue
			}
			return err
		}

		stat, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return err
		}

		if stat.Size() < offset {
			offset = 0
		}
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			return err
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			_, _ = fmt.Fprintln(out, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			_ = f.Close()
			return err
		}
		offset, _ = f.Seek(0, io.SeekCurrent)
		_ = f.Close()

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(300 * time.Millisecond):
		}
	}
}

func followLogFiles(ctx context.Context, out io.Writer, paths []string) error {
	offsets := make(map[string]int64, len(paths))

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			offsets[path] = int64(len(data))
		}
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, path := range paths {
				data, err := os.ReadFile(path)
				if err != nil {
					continue
				}

				offset := offsets[path]
				if int64(len(data)) < offset {
					offset = 0
				}
				if int64(len(data)) == offset {
					continue
				}

				chunk := data[offset:]
				if len(chunk) > 0 {
					_, _ = out.Write(chunk)
				}
				offsets[path] = int64(len(data))
			}
		}
	}
}
