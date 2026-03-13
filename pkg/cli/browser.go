package cli

import (
	"fmt"
	"net"
	"os/exec"
	"runtime"
)

// serverURL derives a browser-navigable URL from a net listen address.
// Wildcard bind addresses ("", "0.0.0.0", "::") are normalised to "localhost"
// so the resulting URL is always reachable from the local machine.
func serverURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// addr without a port – unusual but handle gracefully.
		return "http://" + addr
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "localhost"
	}
	return fmt.Sprintf("http://%s:%s", host, port)
}

// platformOpen opens url in the operating-system's default browser.
// The function returns immediately; the browser runs in a detached process.
func platformOpen(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// "start" is a cmd.exe built-in and must be invoked via cmd /c.
		cmd = exec.Command("cmd", "/c", "start", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default: // linux, bsd, …
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
