package ezghsa

import (
	"errors"

	"github.com/jwalton/gchalk"
)

type SeverityLevel int

const (
	Unknown SeverityLevel = iota
	Low
	Medium
	High
	Critical
)

var ErrInvalidSeverity = errors.New("severity must be low, medium, high, or critical")

func Severity(s string) (SeverityLevel, error) {
	switch s {
	case "":
		return Unknown, nil
	case "low":
		return Low, nil
	case "medium":
		return Medium, nil
	case "high":
		return High, nil
	case "critical":
		return Critical, nil
	default:
		return Unknown, ErrInvalidSeverity
	}
}

func (s SeverityLevel) Abbrev() string {
	switch s {
	case Unknown:
		return gchalk.Dim("??")
	case Low:
		return gchalk.WithBgWhite().WithBlack().Paint("LO")
	case Medium:
		return gchalk.WithBgCyan().WithBlack().Paint("MD")
	case High:
		return gchalk.WithBgYellow().WithBlack().Paint("HI")
	case Critical:
		return gchalk.WithBgRed().WithBlack().Paint("CR")
	default:
		panic(s)
	}
}

func (s SeverityLevel) String() string {
	switch s {
	case Unknown:
		return ""
	case Low:
		return "low"
	case Medium:
		return "medium"
	case High:
		return "high"
	case Critical:
		return "critical"
	default:
		panic(s)
	}
}
