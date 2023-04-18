package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/csdev/ezghsa/internal/ezghsa"
	"github.com/google/go-github/v51/github"
	"github.com/jwalton/gchalk"
	flag "github.com/spf13/pflag"
)

type SeverityValue struct {
	SeverityLevel *ezghsa.SeverityLevel
}

func (v SeverityValue) String() string {
	if v.SeverityLevel != nil {
		return v.SeverityLevel.String()
	}
	return ""
}

func (v SeverityValue) Set(s string) error {
	val, err := ezghsa.Severity(s)
	if err != nil {
		return err
	}

	*v.SeverityLevel = val
	return nil
}

func (v SeverityValue) Type() string {
	return "string"
}

func main() {
	var (
		help    bool
		version bool

		listAll      bool
		failDisabled bool

		severity     ezghsa.SeverityLevel
		failSeverity ezghsa.SeverityLevel
	)

	flag.BoolVarP(&help, "help", "h", help, "display this help text")
	flag.BoolVarP(&version, "version", "V", version, "display version and build info")

	flag.BoolVarP(&listAll, "list-all", "l", listAll, "list all repos that were checked, even those without vulnerabilities")
	flag.BoolVarP(&failDisabled, "fail-disabled", "d", failDisabled, "fail if severity alerts are disabled for a repo")

	flag.VarP(&SeverityValue{&severity}, "severity", "s", "only consider alerts at or above the specified severity level")
	flag.VarP(&SeverityValue{&failSeverity}, "fail-severity", "f", "fail if alerts exist at or above the specified severity level")

	flag.CommandLine.SortFlags = false

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [github organization/repo]\n", os.Args[0])
		flag.PrintDefaults()
	}

	flag.Parse()

	if help {
		flag.Usage()
		return
	}
	if version {
		fmt.Fprintln(os.Stderr, "ezghsa")
		bi, ok := debug.ReadBuildInfo()
		if ok {
			fmt.Fprintf(os.Stderr, "+%v\n", bi)
		} else {
			fmt.Fprintln(os.Stderr, "build information is not available")
		}
		return
	}

	if failSeverity != ezghsa.Unknown && failSeverity < severity {
		fmt.Fprintln(os.Stderr, "conflicting options for \"-f, --fail-severity\" and \"-s, --severity\" flags")
		fmt.Fprintln(os.Stderr, "fail-severity threshold cannot be lower than severity filter")
		os.Exit(1)
	}

	client := ezghsa.Client()

	opts := &github.RepositoryListOptions{
		Affiliation: "owner",
	}
	repos, _, err := client.Repositories.List(context.Background(), "", opts)
	if err != nil {
		log.Fatalf("%v", err)
	}

	worstSeverity := ezghsa.Unknown
	hasDisabled := false

	for _, repo := range repos {
		isEnabled, _, err := client.Repositories.GetVulnerabilityAlerts(
			context.Background(), *repo.Owner.Login, *repo.Name)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if !isEnabled {
			hasDisabled = true
			if listAll || failDisabled {
				fmt.Printf("%s/%s\n%s\n", *repo.Owner.Login, *repo.Name,
					gchalk.Dim("vulnerability alerts are disabled"))
			}
			continue
		}

		opts := &github.ListAlertsOptions{
			State: github.String("open"),
		}
		alerts, _, err := client.Dependabot.ListRepoAlerts(context.Background(), *repo.Owner.Login, *repo.Name, opts)
		if err != nil {
			log.Fatalf("%v", err)
		}

		selectedAlerts := make([]*github.DependabotAlert, 0, len(alerts))

		for _, alert := range alerts {
			adv := alert.SecurityAdvisory
			sev, _ := ezghsa.Severity(adv.GetSeverity())

			if sev < severity {
				continue
			}
			selectedAlerts = append(selectedAlerts, alert)
		}

		if len(selectedAlerts) == 0 {
			if listAll {
				fmt.Printf("%s/%s\n%s\n", *repo.Owner.Login, *repo.Name,
					gchalk.Dim("no matching vulnerability alerts found"))
			}
			continue
		}

		fmt.Printf("%s/%s\n", *repo.Owner.Login, *repo.Name)

		for _, alert := range alerts {
			adv := alert.SecurityAdvisory
			sev, _ := ezghsa.Severity(adv.GetSeverity())

			if sev > worstSeverity {
				worstSeverity = sev
			}

			fmt.Printf("%s %s %s\n", sev.Abbrev(), gchalk.Bold(adv.GetGHSAID()), adv.GetSummary())
		}
	}

	if failSeverity != ezghsa.Unknown && worstSeverity >= failSeverity {
		os.Exit(1 + int(worstSeverity))
	}
	if failDisabled && hasDisabled {
		os.Exit(1)
	}
}
