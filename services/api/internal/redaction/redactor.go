package redaction

import "regexp"

type Finding struct {
	Type  string
	Count int
}
type Redactor struct{ patterns []pattern }
type pattern struct {
	kind string
	re   *regexp.Regexp
}

func New() Redactor {
	return Redactor{patterns: []pattern{
		{"private_key", regexp.MustCompile(`(?s)-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----.*?-----END (?:RSA |EC |OPENSSH )?PRIVATE KEY-----`)},
		{"bearer_token", regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{12,}`)},
		{"jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)},
		{"database_url", regexp.MustCompile(`(?i)\b(?:postgres(?:ql)?|mysql|mongodb(?:\+srv)?|redis)://[^\s]+`)},
		{"password", regexp.MustCompile(`(?i)\b(password|passwd|pwd)\s*[:=]\s*[^\s,;]+`)},
		{"api_key", regexp.MustCompile(`(?i)\b(api[_-]?key|client[_-]?secret|secret|token|private[_-]?key)\s*[:=]\s*["']?[^\s,"';]{8,}`)},
	}}
}

func (r Redactor) Redact(input string) (string, []Finding) {
	out := input
	var findings []Finding
	for _, p := range r.patterns {
		n := len(p.re.FindAllStringIndex(out, -1))
		if n > 0 {
			out = p.re.ReplaceAllString(out, "[REDACTED:"+p.kind+"]")
			findings = append(findings, Finding{p.kind, n})
		}
	}
	return out, findings
}
