package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/BurntSushi/toml"
)

const (
	profileInfoURL = "https://www.googleapis.com/oauth2/v1/userinfo?alt=json"
	oauthLoginUrl  = "https://accounts.google.com/o/oauth2/auth?response_type=code&redirect_uri=%s&client_id=%s&access_type=offline&scope=%s&state=%s"
	sessionName    = "jenkins-oauth"
)

var oauthCfg oauth.Config

var (
	sessionSecret     = flag.String("session-secret", "", "secret session's salt-word")
	sessionPath       = flag.String("session-path", "", "path for cookies")
	sessionTtl        = flag.String("session-ttl", "", "session TTL in seconds")
	sessionHttpOnly   = flag.Bool("session-http-only", true, "http-only cookie")
	oauthScope        = flag.String("oauth-scope", "", "oauth2 scope")
	oauthClientIdNum  = flag.String("oauth-client-id", "", "oauth2 client id")
	oauthClientSecret = flag.String("oauth-secret", "", "oauth secret token")
	oauthCallbackPath = flag.String("oauth-callback", "", "oauth callback relative url")
	dst               = flag.String("jenkins-url", "", "proxy address with Jenkins on other side")
	trustedDomain     = flag.String("trusted-domain", "", "trusted email domain: @domain")
	bind              = flag.String("bind", "", "[host]:port where to serve on")

	// CLI only flags
	configFile = flag.String("config", defaultConfigFile(), "config path")
	Verbose    = flag.Bool("verbose", false, "chatty mode")
)

var conf = AppConfig{
	Oauth: AppOauthConfig{},
	Proxy: AppProxyConfig{},
}

func init() {
	flag.Parse()
	if _, err := toml.DecodeFile(*configFile, &conf); err != nil {
		log.Fatalf("TOML %s parse error: %s\n", *configFile, err.Error())
	}

	/* redefine config options from CLI */
	if *oauthScope != "" {
		conf.Oauth.Scope = *oauthScope
	}
	if *oauthClientIdNum != "" {
		conf.Oauth.ClientId = *oauthClientIdNum
	}
	conf.Oauth.ClientId = fmt.Sprintf("%s.apps.googleusercontent.com", conf.Oauth.ClientId)
	if *oauthClientSecret != "" {
		conf.Oauth.Secret = *oauthClientSecret
	}
	if *oauthCallbackPath != "" {
		conf.Oauth.Callback = *oauthCallbackPath
	}
	if *dst != "" {
		conf.Proxy.JenkinsUrl = *dst
	}
	if *trustedDomain != "" {
		conf.TrustedDomain = *trustedDomain
	}
	if *bind != "" {
		conf.Bind = *bind
	}
	if *sessionSecret != "" {
		conf.Session.Secret = *sessionSecret
	}
	if *sessionPath != "" {
		conf.Session.Path = *sessionPath
	}
	if !*sessionHttpOnly {
		conf.Session.HttpOnly = false
	}
	if *sessionTtl != "" {
		conf.Session.Ttl = *sessionTtl
	}
	dur, err := myParseDuration(conf.Session.Ttl)
	if err != nil {
		log.Fatalf("Error parse session TTL %s: %s\n", conf.Session.Ttl, err.Error())
	}
	conf.Session.TtlSeconds = int(dur.Seconds())

	setupProxyHandler(conf.Proxy.JenkinsUrl)
	setupSessionStore(conf.Session.Secret)

	oauthCfg = oauth.Config{
		ClientId:     conf.Oauth.ClientId,
		ClientSecret: conf.Oauth.Secret,
		Scope:        strings.Replace(conf.Oauth.Scope, "+", " ", -1),
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		RedirectURL:  conf.Oauth.Callback,
	}
}

func myParseDuration(s string) (time.Duration, error) {
	parts := strings.SplitN(s, "d", 2)
	rest := s
	var extraHours int64
	if len(parts) == 2 {
		rest = parts[1]
		hours, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			return time.Duration(0), err
		}
		// precise enough for cookies
		extraHours = int64(hours * 24)
	}

	if rest == "" {
		rest = "0"
	}
	dur, err := time.ParseDuration(rest)
	if err != nil {
		return dur, err
	}

	return time.Duration(int64(dur) + int64(time.Hour*time.Duration(extraHours))), nil
}

type AppConfig struct {
	Oauth         AppOauthConfig
	Proxy         AppProxyConfig
	Session       AppSessionConfig
	Bind          string
	TrustedDomain string   `toml:"trusted_domain"`
	TrustedEmails []string `toml:"trusted_emails"`
}

type AppSessionConfig struct {
	Secret     string
	Ttl        string
	TtlSeconds int
	HttpOnly   bool `toml:"http_only"`
	Path       string
}

type AppProxyConfig struct {
	JenkinsUrl       string   `toml:"jenkins_url"`
	SkipUrls         []string `toml:"skip_auth"`
	DisableUrlDecode bool     `toml:"disable_url_decode"`
}

type AppOauthConfig struct {
	ClientId string `toml:"client_id"`
	Secret   string
	Callback string
	Scope    string
}

type UserInfo struct {
	Email string
	/*
		Id         string
		Name       string
		GivenName  string `json:given_name`
		FamilyName string `json:family_name`
		Link       string
		// https://lh6.googleusercontent.com/-3shaIe0kMJQ/AAAAAAAAAAI/AAAAAAAAAsA/Qd1iADUCr2E/photo.jpg?sz=55
		Picture    string
		Gender     string
		locale     string
	*/
}

func defaultConfigFile() string {
	p, err := build.Default.Import("github.com/nordicdyno/jenkins-goauth2", "", build.FindOnly)
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(p.Dir, "config.toml")
}
