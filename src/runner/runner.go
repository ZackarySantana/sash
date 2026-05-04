package runner

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
)

type Runner interface {
	Run(title string, width, height int, url string) error
}

type StubRunner struct {
	LastURL string
}

func (s *StubRunner) Run(title string, width, height int, url string) error {
	s.LastURL = url
	return nil
}

type Options struct {
	Title       string
	Width       int
	Height      int
	DevURL      string
	Assets      http.FileSystem
	MountAPI    func(mux *http.ServeMux)
	DevAPIMount func(mux *http.ServeMux)
	DevAPIAddr  string
	Runner      Runner
	OpenBrowser bool
}

func Run(opts Options) error {
	if opts.Width <= 0 {
		opts.Width = 1024
	}
	if opts.Height <= 0 {
		opts.Height = 768
	}
	if opts.Title == "" {
		opts.Title = "Sash"
	}

	url := strings.TrimSpace(opts.DevURL)
	var srv *http.Server
	var stopDevAPI func()
	if url != "" {
		if opts.DevAPIMount != nil {
			addr := strings.TrimSpace(os.Getenv("SASH_API_LISTEN"))
			if addr == "" {
				addr = strings.TrimSpace(opts.DevAPIAddr)
			}
			if addr == "" {
				return fmt.Errorf("sash: DevAPIMount requires DevAPIAddr or SASH_API_LISTEN")
			}
			var err error
			_, stopDevAPI, err = startDevAPIServer(addr, opts.DevAPIMount)
			if err != nil {
				return fmt.Errorf("sash: dev API server: %w", err)
			}
			defer stopDevAPI()
		}
	} else {
		if opts.Assets == nil {
			return fmt.Errorf("sash: set Options.DevURL or Options.Assets")
		}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return err
		}
		port := ln.Addr().(*net.TCPAddr).Port
		url = "http://127.0.0.1:" + strconv.Itoa(port)
		mux := http.NewServeMux()
		if opts.MountAPI != nil {
			opts.MountAPI(mux)
		}
		mux.Handle("/", http.FileServer(opts.Assets))
		srv = &http.Server{Handler: mux}
		go func() { _ = srv.Serve(ln) }()
		defer func() {
			_ = srv.Shutdown(context.Background())
		}()
	}

	if opts.Runner != nil {
		return opts.Runner.Run(opts.Title, opts.Width, opts.Height, url)
	}
	return browseUntilInterrupt(url, opts.OpenBrowser)
}

func browseUntilInterrupt(url string, openBrowser bool) error {
	fmt.Fprintf(os.Stderr, "sash: %s (open in browser; Ctrl+C to stop)\n", url)
	if openBrowser {
		if err := openDefaultBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "sash: open browser: %v\n", err)
		}
	}
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	signal.Stop(sigCh)
	return nil
}

func openDefaultBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func startDevAPIServer(addr string, register func(*http.ServeMux)) (baseURL string, stop func(), err error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", nil, err
	}
	mux := http.NewServeMux()
	register(mux)
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	baseURL = "http://" + ln.Addr().String()
	log.Printf("sash: dev API %s", baseURL)
	return baseURL, func() { _ = srv.Close() }, nil
}
