// Command agent-setup is the console setup wizard for Windows (and any OS).
//
// It performs the exact same enrollment + installation as the Fyne GUI
// through the shared SetupController — stdlib only, so it builds anywhere
// without network access or a C toolchain:
//
//	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o dist/windows/VaultGuard-Agent-Setup-Console.exe ./cmd/agent-setup
//
// Run elevated (right-click → Run as administrator); without elevation the
// installer preflight aborts with instructions before touching anything.
// Flags allow non-interactive use; interactive prompts are the default.
// NOTE: console token entry echoes — the token is single-use and never
// stored, but prefer the GUI (masked entry) on shared screens.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/backup-saas/agent/setup"
)

func main() {
	var (
		flagServer = flag.String("server", "", "VaultGuard server URL (e.g. https://vault.example.com/vaultguard/api)")
		flagToken  = flag.String("token", "", "one-time enrollment token (empty = upgrade with existing credential)")
		flagExe    = flag.String("exe", "", "agent binary to install (default: next to this setup exe)")
	)
	flag.Parse()

	fmt.Println("VaultGuard Agent Setup (console)")
	fmt.Println("================================")
	info := setup.DetectMachine()
	fmt.Printf("Machine: %s  Platform: %s/%s  Agent version: %s\n\n",
		nonEmpty(info.MachineName, "unknown"), info.Platform, info.Architecture, info.AgentVersion)

	server := strings.TrimSpace(*flagServer)
	if server == "" {
		server = promptLine("Server URL: ")
	}
	token := *flagToken
	if token == "" && !flagPassed("token") {
		fmt.Println("Enrollment token (one-time, from Dashboard → Agents → Add Agent).")
		fmt.Println("Leave empty to upgrade an existing install (credential preserved).")
		token = promptLine("Token (input echoes — single-use, never stored): ")
	} else if token != "" {
		fmt.Println("WARNING: -token is visible in the process list; prefer interactive entry.")
	}

	exe := *flagExe
	if exe == "" {
		exe = setup.AgentExeNextTo("")
		if exe != "" {
			fmt.Printf("Agent binary: %s\n", exe)
		} else {
			fmt.Println("Agent binary: not found next to setup (binary copy will be skipped).")
		}
	}

	ctl := setup.SetupController{
		Client:    &setup.EnrollmentClient{},
		Installer: setup.NewInstallerForMachine(exe),
		Info:      info,
		OnStep: func(step, detail string) {
			fmt.Printf("[%s] %s\n", step, detail)
		},
	}
	if err := ctl.Run(server, strings.TrimSpace(token)); err != nil {
		fmt.Fprintln(os.Stderr, "Setup failed:", err)
		pressEnterToExit()
		os.Exit(1)
	}
	if ctl.Result != nil {
		fmt.Printf("\nComplete: machine %s, agent %s — service running.\n",
			ctl.Result.MachineName, ctl.Result.AgentID)
	}
	pressEnterToExit()
}

// pressEnterToExit keeps a double-clicked console window open so the result
// (or error) stays readable. With piped/closed stdin it returns instantly,
// so scripting and automation are unaffected.
func pressEnterToExit() {
	fmt.Print("\nPress Enter to exit...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func promptLine(label string) string {
	fmt.Print(label)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(line)
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func flagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}
