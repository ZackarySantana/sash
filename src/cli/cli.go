package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/zackarysantana/sash/src/bindgen"
	"github.com/zackarysantana/sash/src/config"
)

const devServerWait = 90 * time.Second

func Run() error {
	configPath, tags, subcmd, err := parseArgs(os.Args[1:])
	if err != nil {
		return err
	}
	switch subcmd {
	case "version":
		fmt.Println("sash 0.1.0")
		return nil
	case "dev":
		return Dev(configPath, tags)
	case "build":
		return Build(configPath, tags)
	case "bind":
		return Bind(configPath)
	default:
		return fmt.Errorf("unknown command: %s", subcmd)
	}
}

func parseArgs(args []string) (configPath string, tags []string, cmd string, err error) {
	configPath = "sash.json"
	tagSet := map[string]struct{}{}
	addTags := func(s string) {
		for _, p := range strings.Split(s, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				tagSet[p] = struct{}{}
			}
		}
	}

	var cmds []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-help" || a == "--help" || a == "-h":
			return "", nil, "", fmt.Errorf("usage: sash [-config path] [-tags list] <dev|build|bind|version>")
		case a == "-config":
			if i+1 >= len(args) {
				return "", nil, "", fmt.Errorf("-config requires a path")
			}
			configPath = args[i+1]
			i++
		case a == "-tags":
			if i+1 >= len(args) {
				return "", nil, "", fmt.Errorf("-tags requires a value (comma-separated build tags)")
			}
			addTags(args[i+1])
			i++
		case strings.HasPrefix(a, "-tags="):
			addTags(strings.TrimPrefix(a, "-tags="))
		default:
			cmds = append(cmds, a)
		}
	}
	if len(cmds) != 1 {
		return "", nil, "", fmt.Errorf("usage: sash [-config path] [-tags list] <dev|build|bind|version>")
	}
	tags = make([]string, 0, len(tagSet))
	for t := range tagSet {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	return configPath, tags, cmds[0], nil
}

func Dev(configPath string, tags []string) error {
	cfg, baseDir, err := loadResolved(configPath)
	if err != nil {
		return err
	}
	if err := requireGoMain(cfg); err != nil {
		return err
	}
	devURL, port, err := devListen(cfg)
	if err != nil {
		return err
	}

	frontDir := resolvePath(baseDir, cfg.Frontend.Dir)
	front := shellCmd(cfg.Frontend.Dev)
	front.Dir = frontDir
	front.Env = append(os.Environ(), fmt.Sprintf("PORT=%d", port))
	front.Stdout = os.Stdout
	front.Stderr = os.Stderr
	if err := front.Start(); err != nil {
		return err
	}
	defer front.Process.Kill()

	ctx, cancel := context.WithTimeout(context.Background(), devServerWait)
	defer cancel()
	if err := waitDevReady(ctx, devURL, front); err != nil {
		return fmt.Errorf("dev server %s: %w", devURL, err)
	}

	runArgs := append([]string{"run"}, appendGoTags(nil, tags)...)
	runArgs = append(runArgs, cfg.GoMain)

	run := exec.Command("go", runArgs...)
	run.Dir = baseDir
	run.Env = append(os.Environ(), "SASH_DEV_URL="+devURL)
	if cfg.Bindings != nil {
		if a := strings.TrimSpace(cfg.Bindings.DevListenAddr); a != "" {
			run.Env = append(run.Env, "SASH_API_LISTEN="+a)
		}
	}
	run.Stdout = os.Stdout
	run.Stderr = os.Stderr
	return run.Run()
}

func pickFreeListenPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func devListen(cfg *config.Config) (devURL string, port int, err error) {
	const loopback = "127.0.0.1"
	if cfg.Frontend.DevPort > 0 {
		port = cfg.Frontend.DevPort
		return fmt.Sprintf("http://%s:%d", loopback, port), port, nil
	}
	raw := strings.TrimSpace(cfg.Frontend.DevServerURL)
	if raw != "" {
		u, err := url.Parse(raw)
		if err != nil {
			return "", 0, fmt.Errorf("frontend.devServerURL: %w", err)
		}
		host := u.Hostname()
		if host == "" {
			host = loopback
		}
		ps := u.Port()
		if ps == "" {
			return "", 0, fmt.Errorf("frontend.devServerURL must include an explicit port (e.g. http://127.0.0.1:5173)")
		}
		pn, convErr := strconv.Atoi(ps)
		if convErr != nil || pn <= 0 || pn > 65535 {
			return "", 0, fmt.Errorf("frontend.devServerURL has invalid port %q", ps)
		}
		return fmt.Sprintf("http://%s:%d", host, pn), pn, nil
	}
	port, err = pickFreeListenPort()
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port), port, nil
}

func appendGoTags(prefix []string, tags []string) []string {
	if len(tags) == 0 {
		return prefix
	}
	return append(prefix, "-tags="+strings.Join(tags, ","))
}

func Build(configPath string, tags []string) error {
	cfg, baseDir, err := loadResolved(configPath)
	if err != nil {
		return err
	}
	if err := requireGoMain(cfg); err != nil {
		return err
	}
	frontDir := resolvePath(baseDir, cfg.Frontend.Dir)
	if err := shellRun(frontDir, cfg.Frontend.Install); err != nil {
		return err
	}
	if err := shellRun(frontDir, cfg.Frontend.Build); err != nil {
		return err
	}
	outDir := filepath.Join(baseDir, "bin")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	outPath := filepath.Join(outDir, cfg.Name)

	buildArgs := []string{"build", "-o", outPath}
	buildArgs = appendGoTags(buildArgs, tags)
	buildArgs = append(buildArgs, cfg.GoMain)

	cmd := exec.Command("go", buildArgs...)
	cmd.Dir = baseDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Bind(configPath string) error {
	cfg, baseDir, err := loadResolved(configPath)
	if err != nil {
		return err
	}
	if cfg.Bindings == nil {
		return fmt.Errorf("sash.json has no bindings section")
	}
	modRoot, err := findModuleRoot(baseDir)
	if err != nil {
		return err
	}
	return bindgen.Generate(cfg.Bindings, baseDir, modRoot)
}

func findModuleRoot(start string) (string, error) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found above %s", start)
		}
		dir = parent
	}
}

func requireGoMain(cfg *config.Config) error {
	if strings.TrimSpace(cfg.GoMain) == "" {
		return fmt.Errorf("sash.json goMain is required")
	}
	return nil
}

func loadResolved(configPath string) (*config.Config, string, error) {
	absCfg, err := filepath.Abs(configPath)
	if err != nil {
		return nil, "", err
	}
	cfg, err := config.Load(absCfg)
	if err != nil {
		return nil, "", err
	}
	baseDir := filepath.Dir(absCfg)
	return cfg, baseDir, nil
}

func resolvePath(baseDir, p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(baseDir, filepath.Clean(p))
}

func shellCmd(script string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.Command("cmd", "/c", script)
	}
	return exec.Command("sh", "-c", script)
}

func shellRun(dir, script string) error {
	c := shellCmd(script)
	c.Dir = dir
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

func waitDevReady(ctx context.Context, raw string, proc *exec.Cmd) error {
	raw = strings.TrimSpace(raw)

	exitCh := make(chan error, 1)
	go func() {
		exitCh <- proc.Wait()
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for %s: %w", raw, ctx.Err())
		case err := <-exitCh:
			if err != nil {
				return fmt.Errorf("frontend dev command exited before server was ready: %w", err)
			}
			return fmt.Errorf("frontend dev command exited before server was ready")
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, http.NoBody)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			resp.Body.Close()
			if resp.StatusCode < 500 {
				return nil
			}
		}
	}
}
