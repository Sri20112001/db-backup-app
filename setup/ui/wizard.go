// Package ui is the thin Fyne binding over the setup controller
// (github.com/backup-saas/agent/setup). Screens collect input and display
// progress; enrollment, installation, and credential storage all happen in
// the controller. No business logic lives in button callbacks, and no
// secret is ever displayed or logged.
package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	agentsetup "github.com/backup-saas/agent/setup"
)

// Wizard holds the setup state across screens.
type Wizard struct {
	window     fyne.Window
	info       agentsetup.MachineInfo
	serverURL  *widget.Entry
	tokenEntry *widget.Entry
	status     *widget.Label
	progress   *widget.ProgressBar
	busy       bool
	result     *agentsetup.SetupResult
}

// NewWizard builds the wizard with auto-detected machine info, prefilling
// the server URL from a previous install when present (asked once, kept).
func NewWizard(w fyne.Window) *Wizard {
	w.Resize(fyne.NewSize(540, 420))

	server := widget.NewEntry()
	server.SetPlaceHolder("https://vaultguard.example.com/vaultguard/api")
	if saved := agentsetup.SavedServerURL(); saved != "" {
		server.SetText(saved)
	}

	token := widget.NewPasswordEntry()
	token.SetPlaceHolder("Paste enrollment token from dashboard")

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	status.Importance = widget.LowImportance

	return &Wizard{
		window:     w,
		info:       agentsetup.DetectMachine(),
		serverURL:  server,
		tokenEntry: token,
		status:     status,
		progress:   widget.NewProgressBar(),
	}
}

// show wraps screen content in a structured shell (Header, Body, Actions),
// displays it, and returns it for callers that need the object (e.g. the
// initial Welcome content in main).
func (wz *Wizard) show(title, subtitle string, body fyne.CanvasObject, actions []fyne.CanvasObject) fyne.CanvasObject {
	// Top Header
	headerTitle := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	var header fyne.CanvasObject = headerTitle

	if subtitle != "" {
		// Render step indicator as an accent badge
		stepTag := widget.NewRichTextFromMarkdown(fmt.Sprintf("` %s `", subtitle))
		header = container.NewVBox(
			stepTag,
			headerTitle,
		)
	}

	headerContainer := container.NewVBox(
		container.NewPadded(header),
		widget.NewSeparator(),
	)

	// Bottom Action Bar
	var footer fyne.CanvasObject
	if len(actions) > 0 {
		btnRow := container.NewHBox()
		btnRow.Add(layout.NewSpacer())
		for _, act := range actions {
			btnRow.Add(act)
		}
		footer = container.NewVBox(
			widget.NewSeparator(),
			container.NewPadded(btnRow),
		)
	}

	// Padded central container
	contentShell := container.NewBorder(
		headerContainer,
		footer,
		nil,
		nil,
		container.NewPadded(body),
	)

	wz.window.SetContent(contentShell)
	return contentShell
}

// Welcome is screen 1.
func (wz *Wizard) Welcome() fyne.CanvasObject {
	introText := widget.NewRichTextFromMarkdown(
		"Welcome to **VaultGuard Agent** setup. This installer will register your computer " +
			"with your organization's backup cloud and configure the background service.",
	)
	introText.Wrapping = fyne.TextWrapWord

	sysInfoCard := widget.NewCard(
		"Detected System Details",
		"",
		container.NewVBox(
			container.NewGridWithColumns(2,
				widget.NewLabelWithStyle("Computer Name:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(nonEmpty(wz.info.MachineName, "This Machine")),
				widget.NewLabelWithStyle("Operating System:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("%s (%s)", wz.info.Platform, wz.info.Architecture)),
				widget.NewLabelWithStyle("Agent Version:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				widget.NewLabel(fmt.Sprintf("v%s", wz.info.AgentVersion)),
			),
		),
	)

	body := container.NewVBox(
		introText,
		layout.NewSpacer(),
		sysInfoCard,
		layout.NewSpacer(),
	)

	actions := []fyne.CanvasObject{
		widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { wz.window.Close() }),
		widget.NewButtonWithIcon("Get Started", theme.NavigateNextIcon(), func() { wz.Connect() }),
	}

	return wz.show("VaultGuard Agent Setup", "Step 1 of 3: Verification", body, actions)
}

// Connect is screen 2: server URL + enrollment token.
func (wz *Wizard) Connect() {
	wz.status.SetText("")

	form := widget.NewForm(
		widget.NewFormItem("Server URL", wz.serverURL),
		widget.NewFormItem("Enrollment Token", wz.tokenEntry),
	)

	helpText := widget.NewLabel("Single-use token — generate it in Agents → Add Agent.")
	helpText.Wrapping = fyne.TextWrapWord
	helpText.Importance = widget.LowImportance

	body := container.NewVBox(
		form,
		helpText,
		layout.NewSpacer(),
		wz.status,
	)

	backBtn := widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), func() { wz.Welcome() })
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { wz.window.Close() })
	connectBtn := widget.NewButtonWithIcon("Enroll & Install", theme.ConfirmIcon(), func() { wz.register() })
	connectBtn.Importance = widget.HighImportance

	actions := []fyne.CanvasObject{backBtn, cancelBtn, connectBtn}
	wz.show("Connect to VaultGuard", "Step 2 of 3: Authentication", body, actions)
}

// register is screen 3: progress while the controller works.
func (wz *Wizard) register() {
	if wz.busy {
		return
	}

	server := wz.serverURL.Text
	token := wz.tokenEntry.Text

	if err := agentsetup.ValidateServerURL(server); err != nil {
		wz.showError("Invalid Server URL", err.Error())
		return
	}
	if token == "" {
		wz.showError("Token Required", "Paste the one-time enrollment token from Agents → Add Agent, or generate one first.")
		return
	}

	wz.tokenEntry.SetText("")
	wz.busy = true
	wz.progress.SetValue(0)
	wz.status.SetText("Contacting enrollment endpoint…")

	body := container.NewVBox(
		layout.NewSpacer(),
		widget.NewLabelWithStyle("Configuring service and registering agent…", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		wz.progress,
		container.NewCenter(wz.status),
		layout.NewSpacer(),
	)

	// No action buttons during registration to prevent interruptions
	wz.show("Enrolling Machine", "Step 3 of 3: Installation", body, nil)

	go func() {
		ctl := agentsetup.SetupController{
			Client:    &agentsetup.EnrollmentClient{},
			Installer: agentsetup.NewInstallerForMachine(agentExeNextToSetup()),
			Info:      wz.info,
		}

		steps := 0
		ctl.OnStep = func(step, detail string) {
			steps++
			wz.status.SetText(detail)
			wz.progress.SetValue(float64(steps) / 8.0)

			if step == "done" {
				wz.result = ctl.Result
				wz.complete()
			}
		}

		if err := ctl.Run(server, token); err != nil {
			wz.busy = false
			wz.showError("Installation Failed", err.Error())
		}
	}()
}

// complete is the final screen: success summary + Finish.
func (wz *Wizard) complete() {
	agentID := "—"
	if wz.result != nil && wz.result.AgentID != "" {
		agentID = wz.result.AgentID
	}

	summaryCard := widget.NewCard(
		"Registration Complete",
		"The service has been registered and is now running.",
		container.NewGridWithColumns(2,
			widget.NewLabelWithStyle("Computer:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(nonEmpty(wz.info.MachineName, "—")),
			widget.NewLabelWithStyle("Agent ID:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel(agentID),
			widget.NewLabelWithStyle("Service State:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Running"),
		),
	)

	body := container.NewVBox(
		layout.NewSpacer(),
		summaryCard,
		layout.NewSpacer(),
	)

	finishBtn := widget.NewButtonWithIcon("Finish", theme.ConfirmIcon(), func() {
		wz.window.Close()
	})
	finishBtn.Importance = widget.HighImportance

	wz.show("Setup Succeeded", "Installation complete", body, []fyne.CanvasObject{finishBtn})
}

// showError renders failures as a proper wizard screen (not a popup): a
// readable error card with a single way back. Messages come from the
// controller/sanitized validators, so they never contain secrets. The text
// scrolls inside a fixed area so long errors are never cut off.
func (wz *Wizard) showError(title, message string) {
	msg := widget.NewLabel(message)
	msg.Wrapping = fyne.TextWrapWord

	scroll := container.NewVScroll(msg)
	scroll.SetMinSize(fyne.NewSize(420, 140))

	card := widget.NewCard(
		title,
		"What you can do next is below.",
		scroll,
	)

	body := container.NewVBox(
		layout.NewSpacer(),
		card,
		layout.NewSpacer(),
	)

	backBtn := widget.NewButtonWithIcon("Try Again", theme.NavigateBackIcon(), func() { wz.Connect() })
	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), func() { wz.window.Close() })

	wz.show("Setup Encountered a Problem", "Action required", body, []fyne.CanvasObject{backBtn, cancelBtn})
}

func nonEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

func agentExeNextToSetup() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	for _, name := range []string{"VaultGuard-Agent.exe", "vaultguard-agent", "vaultguard-agent.exe"} {
		candidate := filepath.Join(filepath.Dir(exe), name)
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return ""
}
