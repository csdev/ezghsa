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
		fail         bool

		ghsa     string
		cve      string
		severity ezghsa.SeverityLevel
		days     int
		closed   bool

		ownerRepoNames []string
	)

	flag.BoolVarP(&help, "help", "h", help, "display this help text")
	flag.BoolVarP(&version, "version", "V", version, "display version and build info")

	flag.BoolVarP(&listAll, "list-all", "l", listAll, "list all repos that were checked, even those without vulnerabilities")
	flag.BoolVarP(&failDisabled, "fail-disabled", "D", failDisabled, "fail if severity alerts are disabled for a repo")
	flag.BoolVarP(&fail, "fail", "F", fail, "fail if matching alerts are found")

	// filter options
	flag.StringVarP(&ghsa, "ghsa", "g", ghsa, "filter alerts by GHSA ID")
	flag.StringVarP(&cve, "cve", "c", cve, "filter alerts by CVE ID")
	flag.VarP(&SeverityValue{&severity}, "severity", "s", "filter to alerts at or above the specified severity level")
	flag.IntVarP(&days, "days", "d", days, "filter to alerts older than the specified number of days")
	flag.BoolVar(&closed, "closed", closed, "include closed alerts")

	flag.StringSliceVarP(&ownerRepoNames, "repo", "r", ownerRepoNames, "comma-separated list of repos to check, in OWNER/REPO format")

	flag.CommandLine.SortFlags = false

	flag.Usage = func() {
		const usage = "Usage: %s [options]\n" +
			"       %s [options] --repo OWNER/REPO[,...]\n"

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
	alertCount := 0

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

		var alerts []*github.DependabotAlert
		if closed {
			alerts, err = app.GetAllAlerts(ownerName, repoName)
		} else {
			alerts, err = app.GetOpenAlerts(ownerName, repoName)
		}

		if err != nil {
			log.Fatalf("%v", err)
		}

		selectedAlerts := ezghsa.FilterAlerts(alerts, func(a *github.DependabotAlert) bool {
			if ghsa != "" && a.SecurityAdvisory.GetGHSAID() != ghsa {
				return false
			}
			if cve != "" && a.SecurityAdvisory.GetCVEID() != cve {
				return false
			}

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

		for _, alert := range selectedAlerts {
			adv := alert.SecurityAdvisory
			sev, _ := ezghsa.Severity(adv.GetSeverity())
			dur := now.Sub(alert.CreatedAt.Time)

			summary := adv.GetSummary()
			cveid := adv.GetCVEID()
			if cveid != "" {
				summary = fmt.Sprintf("%s (%s)", summary, cveid)
			}

			state := alert.GetState()
			if state != "" && state != "open" {
				// de-emphasize closed alerts
				summary = gchalk.Dim(fmt.Sprintf("%s: %s", state, summary))
			} else {
				// count open alerts
				alertCount++
				if sev > worstSeverity {
					worstSeverity = sev
				}
			}

			fmt.Printf("%s %s %s %s\n", sev.Abbrev(), gchalk.Bold(adv.GetGHSAID()), DayAbbrev(&dur), summary)
		}
	}

	if fail && alertCount > 0 {
		os.Exit(1 + int(worstSeverity))
	}
	if failDisabled && hasDisabled {
		os.Exit(1)
	}
}
