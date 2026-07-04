package headers

import "strings"

// Device holds coarse client info derived from the User-Agent, used to
// enrich audit and login-history records.
type Device struct {
	Browser   string
	OS        string
	UserAgent string
}

// ParseDevice does lightweight User-Agent sniffing (no external dependency).
// It is best-effort: audit value, not security-critical.
func ParseDevice(userAgent string) Device {
	ua := strings.ToLower(userAgent)
	return Device{
		Browser:   detectBrowser(ua),
		OS:        detectOS(ua),
		UserAgent: userAgent,
	}
}

func detectBrowser(ua string) string {
	switch {
	case strings.Contains(ua, "edg/"):
		return "Edge"
	case strings.Contains(ua, "opr/"), strings.Contains(ua, "opera"):
		return "Opera"
	case strings.Contains(ua, "chrome/"):
		return "Chrome"
	case strings.Contains(ua, "firefox/"):
		return "Firefox"
	case strings.Contains(ua, "safari/"):
		return "Safari"
	case strings.Contains(ua, "curl/"):
		return "curl"
	case ua == "":
		return "unknown"
	default:
		return "other"
	}
}

func detectOS(ua string) string {
	switch {
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "iphone"), strings.Contains(ua, "ipad"):
		return "iOS"
	case strings.Contains(ua, "mac os"), strings.Contains(ua, "macintosh"):
		return "macOS"
	case strings.Contains(ua, "linux"):
		return "Linux"
	default:
		return "unknown"
	}
}
