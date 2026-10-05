// Command VaultGuard-Agent-Setup is the Fyne-based setup GUI.
//
// It is ONLY the installation UI: it collects the server URL + one-time
// enrollment token, registers this machine via the enrollment API, writes
// non-secret config, stores the permanent credential, and installs/starts
// the headless VaultGuard Agent (Windows Service / systemd). After Finish
// it exits; backups are performed by the agent, never by this program.
//
// Build (requires network for module downloads + a C toolchain for Fyne):
//
//	go build -trimpath -ldflags "-H=windowsgui" -o VaultGuard-Agent-Setup.exe .
//
// -H=windowsgui sets the Windows GUI subsystem so no console window opens
// beside the app. (The console setup in agent/cmd/agent-setup intentionally
// keeps the default console subsystem.)
// Cross-compile the Windows GUI from Linux with a Windows build agent
// (Fyne needs platform graphics backends); see README.md.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/backup-saas/setup/ui"
)

func main() {
	a := app.NewWithID("com.vaultguard.setup")
	// VaultGuard brand palette (warm paper light / Cyberdeck Night dark).
	a.Settings().SetTheme(&ui.VaultGuardTheme{})
	w := a.NewWindow("VaultGuard Agent Setup")
	w.Resize(fyne.NewSize(540, 660))
	w.SetContent(ui.NewWizard(w).Welcome())
	w.ShowAndRun()
}
