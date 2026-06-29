package contextref

import (
	"fmt"
	"net/url"
	"strings"
)

func Encode(tenant, session, object string) string {
	return fmt.Sprintf("ctx://%s/%s/%s", url.PathEscape(tenant), url.PathEscape(session), url.PathEscape(object))
}
func Parse(ref string) (tenant, session, object string, err error) {
	u, e := url.Parse(ref)
	if e != nil || u.Scheme != "ctx" || u.Host == "" {
		return "", "", "", fmt.Errorf("invalid context ref")
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 {
		return "", "", "", fmt.Errorf("invalid context ref")
	}
	return u.Host, parts[0], parts[1], nil
}
