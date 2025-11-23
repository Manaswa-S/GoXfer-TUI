package auxiliaryui

// import (
// 	"github.com/rivo/tview"
// )

// type Help struct {
// 	app     *tview.Application
// 	flex    *tview.Flex
// 	updater *Updater

// 	help *tview.Form
// }

// func newHelp(app *tview.Application, updater *Updater) *Help {
// 	return &Help{
// 		app:     app,
// 		flex:    tview.NewFlex().SetDirection(tview.FlexRow),
// 		updater: updater,
// 	}
// }

// func (a *Help) initHelp() {
// 	defer a.setFlex()

// 	a.help = tview.NewForm().
// 		AddTextView("Help:", helpContent, 0, 0, true, true).
// 		AddButton("Back", func() {
// 			a.updater.switchPage(PAGE_MENU)
// 		})
// 	a.help.SetBorder(true).SetTitle("Help").SetTitleAlign(tview.AlignLeft)
// }

// func (a *Help) setFlex() {
// 	a.flex.AddItem(a.help, 0, 1, true)
// }

// const helpContent string = `=============================
// GoXfer - Help & Documentation
// =============================

// Welcome to GoXfer, a secure, minimal, and stateless file transfer and
// storage service. This help section explains how GoXfer works, how to
// navigate through the interface, and how to perform basic operations
// such as creating, finding, and using buckets.

// ----------------------------------------------------------------------
// 1. Overview
// ----------------------------------------------------------------------
// GoXfer allows you to transfer or store files inside isolated entities
// called "Buckets". Each bucket is protected by a password and identified
// by a unique bucket key. GoXfer is fully stateless — which means it
// does not store user sessions or credentials anywhere.

// All access is derived from the bucket password you provide at the time
// of performing an operation. If you lose your bucket key or password,
// the data inside that bucket becomes permanently inaccessible.

// ----------------------------------------------------------------------
// 2. Buckets
// ----------------------------------------------------------------------
// A bucket acts as a small logical container. Every bucket holds files
// and related metadata. You can create multiple buckets, each with its
// own password and encryption scope.

// When you create a new bucket:
//   - A unique Bucket Key is generated.
//   - You must remember and store this key safely.
//   - The password you choose will be used to encrypt data inside it.
//   - No one, including the server, can recover it if lost.

// When you access (or "find") a bucket:
//   - You must provide the correct Bucket Key.
//   - You must also provide the correct password.
//   - If both are valid, GoXfer derives the encryption key for access.

// ----------------------------------------------------------------------
// 3. Using the Interface
// ----------------------------------------------------------------------
// GoXfer is a text-based user interface (TUI) application.
// You can navigate the interface using your keyboard.

// General key bindings:
//   - Arrow Up / Down  : Move between items or fields
//   - Tab / Shift+Tab  : Switch focus between input fields
//   - Enter            : Confirm a choice or activate a button
//   - Esc or Q         : Return to the previous menu or quit

// Menu navigation example:
//   1. Find Bucket
//   2. Create Bucket
//   3. Help
//   4. About
//   5. Quit

// You can select an option by using the arrow keys and pressing Enter.

// ----------------------------------------------------------------------
// 4. Creating a Bucket
// ----------------------------------------------------------------------
// To create a new bucket:
//   1. Choose the "Create Bucket" option from the entry menu.
//   2. Enter a desired bucket name and password.
//   3. Confirm the action.
//   4. The system will display a new Bucket ID once created.

// Be sure to note down the Bucket ID and password immediately.
// These two pieces of information are required for all future access.

// If you lose either one, there is no recovery mechanism.
// GoXfer does not maintain user records or copies of credentials.

// ----------------------------------------------------------------------
// 5. Finding an Existing Bucket
// ----------------------------------------------------------------------
// To access an existing bucket:
//   1. Choose the "Find Bucket" option from the entry menu.
//   2. Enter the Bucket ID and password.
//   3. If both match, the bucket will be unlocked for this session.
//   4. You can then perform operations such as listing or adding files.

// If incorrect details are entered, an error message will appear.
// Ensure that the ID and password are exactly as originally assigned.

// ----------------------------------------------------------------------
// 6. File Operations
// ----------------------------------------------------------------------
// Once inside an unlocked bucket, you can:
//   - Upload files
//   - List existing files
//   - Download files
//   - Delete files
//   - View metadata

// Each of these operations is executed independently and requires
// re-authentication with the bucket password each time. The system
// remains stateless between actions to ensure security and isolation.

// Data integrity and encryption are handled transparently in the
// background. Files are encrypted using a key derived from the password
// and salt associated with that specific bucket.

// ----------------------------------------------------------------------
// 7. Security Principles
// ----------------------------------------------------------------------
// GoXfer is designed with the following principles:

//   - Statelessness:
//       No sessions or cookies are stored. Each operation stands alone.

//   - Encryption by Design:
//       Data encryption occurs automatically using the provided password.

//   - Zero Knowledge:
//       The server never learns your password or your encryption keys.

//   - Irrecoverable Loss:
//       Losing the bucket password or ID results in permanent data loss.

//   - Local Responsibility:
//       It is your responsibility to manage and back up keys securely.

// ----------------------------------------------------------------------
// 8. Troubleshooting
// ----------------------------------------------------------------------
// Common issues and their solutions:

//   - Problem: "Invalid bucket ID or password"
//     Solution: Ensure there are no leading/trailing spaces and try again.

//   - Problem: "Connection failed"
//     Solution: Check your network and server status.

//   - Problem: Application layout is broken or unreadable
//     Solution: Resize your terminal window to at least 100x30 characters.

//   - Problem: Scrolling not working in help or log views
//     Solution: Use Arrow Up/Down or Page Up/Down to scroll.

//   - Problem: Keys not responding
//     Solution: Focus may be on a different element; press Tab to cycle.

// ----------------------------------------------------------------------
// 9. Tips and Notes
// ----------------------------------------------------------------------
// - Each bucket is independent. Actions on one bucket do not affect others.
// - Remember that "unlocking" a bucket is not a login; it only verifies
//   access credentials for that single operation.
// - The server stores no user profile, preferences, or session state.
// - Buckets cannot be renamed once created.
// - For best results, avoid using spaces or special symbols in bucket names.

// ----------------------------------------------------------------------
// 10. Development Notes
// ----------------------------------------------------------------------
// GoXfer is written in Go and uses the tview library for its interface.
// It follows a layered architecture separating the TUI layer, service
// layer, and backend communication layer. The service layer handles all
// interactions with the backend, while the TUI manages display and user
// input.

// Design goals:
//   - Clear separation of UI and logic
//   - Stateless communication
//   - Secure encryption at rest and in transit
//   - Lightweight, fast, and portable across systems

// ----------------------------------------------------------------------
// 11. Support and Contact
// ----------------------------------------------------------------------
// For bug reports, feedback, or suggestions, please contact the project
// maintainers through the official repository or support address.

// Example (placeholder):
//   Repository: https://github.com/example/goxfer
//   Support:    support@goxfer.local

// ----------------------------------------------------------------------
// End of Help File
// ----------------------------------------------------------------------
// Press any key to return to the menu.
// `
