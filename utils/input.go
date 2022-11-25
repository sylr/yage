package utils

import (
	"fmt"
	"os"
	"runtime"

	"golang.org/x/term"
)

// ReadPassphrase reads a passphrase from the terminal. It does not read from a
// non-terminal stdin, so it does not check stdinInUse.
func ReadPassphrase(prompt string) ([]byte, error) {
	var in, out *os.File
	if runtime.GOOS == "windows" {
		var err error
		in, err = os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		defer in.Close()
		out, err = os.OpenFile("CONOUT$", os.O_WRONLY, 0)
		if err != nil {
			return nil, err
		}
		defer out.Close()
	} else if _, err := os.Stat("/dev/tty"); err == nil {
		tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		defer tty.Close()
		in, out = tty, tty
	} else {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, fmt.Errorf("standard input is not a terminal, and /dev/tty is not available: %v", err)
		}
		in, out = os.Stdin, os.Stderr
	}
	fmt.Fprintf(out, "%s ", prompt)
	// Use CRLF to work around an apparent bug in WSL2's handling of CONOUT$.
	// Only when running a Windows binary from WSL2, the cursor would not go
	// back to the start of the line with a simple LF. Honestly, it's impressive
	// CONIN$ and CONOUT$ even work at all inside WSL2.
	defer fmt.Fprintf(out, "\r\n")
	return term.ReadPassword(int(in.Fd()))
}
