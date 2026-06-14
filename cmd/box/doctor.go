package box

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/shariqnaiyer/agentbox/internal/doctorcheck"
	"github.com/shariqnaiyer/agentbox/internal/platform"
)

func cmdDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "JSON output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	checks := doctorcheck.RunAll(platform.Detect())

	if *asJSON {
		b, _ := json.MarshalIndent(checks, "", "  ")
		fmt.Println(string(b))
	} else {
		for _, c := range checks {
			mark := mark(c.Severity, c.OK)
			fmt.Printf("%s %-18s %s\n", mark, c.Name, c.Detail)
			if !c.OK && c.Fix != "" {
				fmt.Printf("                      ↳ %s\n", c.Fix)
			}
		}
	}

	// Exit non-zero if any error-severity check failed (the CI/smoke gate).
	if fails := doctorcheck.Failures(checks); len(fails) > 0 {
		fmt.Fprintf(os.Stderr, "\n%d critical issue(s).\n", len(fails))
		os.Exit(1)
	}
	return nil
}

func mark(sev doctorcheck.Severity, okv bool) string {
	if okv {
		return "✓"
	}
	if sev == doctorcheck.SevError {
		return "✗"
	}
	return "!"
}
