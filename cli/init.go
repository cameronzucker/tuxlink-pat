package cli

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/la5nta/pat/app"
	"github.com/la5nta/pat/cfg"

	"github.com/pd0mz/go-maidenhead"
)

// printWizardRedirect prints the credential-entry redirect message and exits
// cleanly. Per spec §4.4: tuxlink-pat is tuxlink's engine, NOT a standalone
// Pat replacement; the wizard owns credential entry. Standalone-Pat users
// should use upstream la5nta/pat which retains config.json passwords.
//
// Returns nothing because it terminates the process.
func printWizardRedirect() {
	fmt.Println(`Skipping credential setup — tuxlink-pat does not collect Winlink credentials.
Set credentials via the tuxlink wizard (writes to OS keyring).
For standalone Pat usage, use upstream la5nta/pat which retains config.json passwords.
See: https://github.com/cameronzucker/tuxlink-pat (README Credentials section)`)
	os.Exit(0)
}

func InitHandle(ctx context.Context, a *app.App, args []string) {
	cancel := exitOnContextCancellation(ctx)
	defer cancel()

	cfgPath := a.Options().ConfigPath
	cfg, err := app.LoadConfig(cfgPath, cfg.DefaultConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Pat Initial Configuration")
	fmt.Println("=========================")
	fmt.Print("(Press ctrl+c at any time to abort)\n\n")

	// Prompt for callsign
	callsign := prompt("Enter your callsign", cfg.MyCall)
	if callsign == "" {
		log.Fatal("Callsign is required")
	}
	cfg.MyCall = strings.ToUpper(callsign)

	// Prompt for Maidenhead grid square
	locator := prompt("Enter your Maidenhead locator", cfg.Locator)
	if locator == "" {
		log.Fatal("Maidenhead locator is required")
	}
	if _, err := maidenhead.ParseLocator(locator); err != nil {
		fmt.Printf("⚠ %q might be an invalid locator. Using it anyway.\n", locator)
	}
	cfg.Locator = locator

	// Persist non-credential config BEFORE redirecting (R1 F5: no half-configured
	// state — whatever the user typed for callsign/locator/mailbox is saved). The
	// credential setup that upstream pat handled here is now owned by the
	// tuxlink wizard, which writes credentials to the OS keyring rather than to
	// config.json. Both new-account and existing-account paths route through
	// the same exit; there is no longer a meaningful branch on accountExists.
	if err := app.WriteConfig(cfg, cfgPath); err != nil {
		log.Fatalf("Failed to write config: %v", err)
	}
	printWizardRedirect()
	// unreachable; printWizardRedirect calls os.Exit(0).
}
