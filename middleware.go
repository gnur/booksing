package main

import (
	"context"
	"net/http"
	"runtime"

	"github.com/coreos/go-oidc"
)

var (
	timeFormat = "02/Jan/2006:15:04:05 -0700"
	authDomain = "https://egdk.cloudflareaccess.com"
	certsUrl   = "https://egdk.cloudflareaccess.com/cdn-cgi/access/certs"
	//maybe this is private? I don't know, I feel pretty safe committing this for now
	audience = "5f08d8662487c020cb66fc58f85e1189e848112088c1de0ab60236f264098db0"
)

type claims struct {
	Email string `json:"email"`
	CN    string `json:"common_name"`
	Type  string `json:"type"`
}

func verifyJWT(cfHeader string) (string, bool) {
	ctx := context.TODO()

	config := &oidc.Config{
		ClientID: audience,
	}
	keyset := oidc.NewRemoteKeySet(ctx, certsUrl)
	verifier := oidc.NewVerifier(authDomain, keyset, config)
	t, err := verifier.Verify(ctx, cfHeader)
	if err != nil {
		return "", false
	}
	var c claims
	err = t.Claims(&c)
	id := ""
	if err == nil {
		//maybe this should be exclusive?
		if c.CN != "" {
			id = c.CN
		}
		if c.Email != "" {
			id = c.Email
		}
	}
	return id, true
}

func getUserFromRequest(r *http.Request) string {
	user := "unknown"
	if runtime.GOOS == "darwin" {
		user = "erwin-dev"
	}

	if r.Header.Get("X-Tobab-User") != "" {
		user = r.Header.Get("X-Tobab-User")
	}

	if r.Header.Get("Tailscale-User-Login") != "" {
		user = r.Header.Get("Tailscale-User-Login")
	}

	id, ok := verifyJWT(r.Header.Get("Cf-Access-Jwt-Assertion"))
	if ok {
		user = id
	}
	return user
}

var IPHeaders = []string{
	"X-Real-IP",
	"X-Forwarded-For",
	"X-Appengine-Remote-Addr",
	"CF-Connecting-IP",
	"Fastly-Client-IP",
	"True-Client-IP",
	"x-original-forwarded-for",
}

func getIPFromRequest(r *http.Request) []string {
	ips := []string{}

	ips = append(ips, r.RemoteAddr)

	for _, header := range IPHeaders {
		ip := r.Header.Get(header)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}
