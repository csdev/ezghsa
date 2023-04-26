package main

import (
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"

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

func DayAbbrev(dur *time.Duration) string {
	days := dur.Hours() / 24
	s := fmt.Sprintf("%3dd", int(days))
	if days < 0 {
		return gchalk.Dim(s)
	} else if days < 7 {
		return gchalk.Blue(s)
	} else if days < 30 {
		return gchalk.Yellow(s)
	} else {
		return gchalk.Red(s)
	}
}

func main() {
	var (
		help    bool
		version bool

		listAll      bool
		failDisabled bool

		severity     ezghsa.SeverityLevel
		failSeverity ezghsa.SeverityLevel

		days     int
		failDays int

		ownerRepoNames []string
	)

	flag.BoolVarP(&help, "help", "h", help, "display this help text")
	flag.BoolVarP(&version, "version", "V", version, "display version and build info")

	flag.BoolVarP(&listAll, "list-all", "l", listAll, "list all repos that were checked, even those without vulnerabilities")
	flag.BoolVarP(&failDisabled, "fail-disabled", "d", failDisabled, "fail if severity alerts are disabled for a repo")

	flag.VarP(&SeverityValue{&severity}, "severity", "s", "only consider alerts at or above the specified severity level")
	flag.VarP(&SeverityValue{&failSeverity}, "fail-severity", "f", "fail if alerts exist at or above the specified severity level")

	flag.IntVarP(&days, "age", "a", days, "only consider alerts older than the specified number of days")
	flag.IntVarP(&failDays, "fail-age", "A", failDays, "fail if alerts are older than the specified number of days")

	flag.StringSliceVarP(&ownerRepoNames, "repos", "r", ownerRepoNames, "comma-separated list of repos to check, in OWNER/REPO format")

	flag.CommandLine.SortFlags = false

	flag.Usage = func() {
		const usage = "Usage: %s [options]\n" +
			"       %s [options] --repos OWNER/REPO[,...]\n"

		fmt.Fprintf(os.Stderr, usage, os.Args[0], os.Args[0])
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

	if flag.NArg() > 0 {
		flag.Usage()
		os.Exit(1)
	}

	if failSeverity != ezghsa.Unknown && failSeverity < severity {
		fmt.Fprintln(os.Stderr, "conflicting options for \"-f, --fail-severity\" and \"-s, --severity\" flags")
		fmt.Fprintln(os.Stderr, "fail-severity threshold cannot be lower than severity filter")
		os.Exit(1)
	}
	if days != 0 && failDays != 0 && days > failDays {
		fmt.Fprintln(os.Stderr, "conflicting options for \"-a, --age\" and \"-A --fail-age\" flags")
		fmt.Fprintln(os.Stderr, "failure age cannot be lower than age filter")
		os.Exit(1)
	}

	app := ezghsa.New(ezghsa.DefaultHttpClient())

	var repos []*github.Repository
	if len(ownerRepoNames) > 0 {
		var err error
		repos, err = app.GetRepos(ownerRepoNames)
		if err != nil {
			log.Fatalf("error getting repositories: %v", err)
		}
	} else {
		var err error
		repos, err = app.GetMyRepos()
		if err != nil {
			log.Fatalf("error getting repositories for the current user: %v", err)
		}
	}

	now := time.Now().UTC()
	hasDisabled := false
	worstSeverity := ezghsa.Unknown
	oldestCreated := now

	cutoffTime := now
	if days != 0 {
		cutoffTime = now.AddDate(0, 0, -days)
	}

	for _, repo := range repos {
		ownerName := repo.GetOwner().GetLogin()
		repoName := repo.GetName()

		isEnabled, err := app.CheckAlertsEnabled(ownerName, repoName)
		if err != nil {
			log.Fatalf("%v", err)
		}
		if !isEnabled {
			hasDisabled = true
			if listAll || failDisabled {
				fmt.Printf("%s/%s\n%s\n", ownerName, repoName,
					gchalk.Dim("vulnerability alerts are disabled"))
			}
			continue
		}

		alerts, err := app.GetOpenAlerts(ownerName, repoName)
		if err != nil {
			log.Fatalf("%v", err)
		}

		selectedAlerts := ezghsa.FilterAlerts(alerts, func(a *github.DependabotAlert) bool {
			sev, _ := ezghsa.Severity(a.SecurityAdvisory.GetSeverity())
			return sev >= severity && a.CreatedAt.Time.Before(cutoffTime)
		})

		if len(selectedAlerts) == 0 {
			if listAll {
				fmt.Printf("%s/%s\n%s\n", ownerName, repoName,
					gchalk.Dim("no matching vulnerability alerts found"))
			}
			continue
		}

		fmt.Printf("%s/%s\n", ownerName, repoName)

		for _, alert := range alerts {
			adv := alert.SecurityAdvisory
			sev, _ := ezghsa.Severity(adv.GetSeverity())
			dur := now.Sub(alert.CreatedAt.Time)

			if sev > worstSeverity {
				worstSeverity = sev
			}
			if alert.CreatedAt.Time.Before(oldestCreated) {
				oldestCreated = alert.CreatedAt.Time
			}

			fmt.Printf("%s %s %s %s\n", sev.Abbrev(), gchalk.Bold(adv.GetGHSAID()), DayAbbrev(&dur),
				adv.GetSummary())
		}
	}

	if failSeverity != ezghsa.Unknown && worstSeverity >= failSeverity {
		os.Exit(1 + int(worstSeverity))
	}
	if failDays != 0 && oldestCreated.Before(now.AddDate(0, 0, -failDays)) {
		os.Exit(1)
	}
	if failDisabled && hasDisabled {
		os.Exit(1)
	}
}
