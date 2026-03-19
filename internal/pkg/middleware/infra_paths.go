package middleware

func IsInfraPath(p string) bool {
	switch {
	case p == "/healthz/live",
		p == "/healthz/ready",
		p == "/metrics",
		len(p) >= 8 && p[:8] == "/swagger":
		return true
	}

	return false
}
