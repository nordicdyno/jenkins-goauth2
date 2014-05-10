jenkins-goauth2
================

`jenkins-goauth2` is Jenkins & Google OAuth2 Security Proxy

written in Go so it's just single binary + config file

## Features

* allow configurable access from Google OAuth accounts
* allow urls without auth
* expvar handler

## Getting jenkins-goauth2

The latest release is available as a binary at [Github][github-release]

[github-release]: https://github.com/nordicdyno/jenkins-goauth2/releases/


You can also build jenkins-goauth2 from source:

`go get github.com/nordicdyno/jenkins-goauth2/`

## How to run

```
sudo ./jenkins-goauth2 -config config.toml
```

### config.toml example

Toml format description: https://github.com/mojombo/toml

```toml
bind = ":80"
trusted_domain = "@sports.ru"
# trusted_emails - toml only accessible (no CLI option)
trusted_emails = [
	"nordicdyno@gmail.com",
]

[session]
secret = "sports.ru"
# Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h", "d"
ttl = "7d"
http_only = true
path = "/"

[proxy]
jenkins_url  = "http://jenkins-url:8080"
# skip_auth - toml only accessible (no CLI option)
# skip_auth - are urls with Gorilla Muxer syntax: http://www.gorillatoolkit.org/pkg/mux
skip_auth = [
# From Jenkins doc
	"/bitbucket-hook{rest:.*}",
	"/cli{rest:.*}",
	"/git{rest:.*}",
	"/jnlpJars{rest:.*}",
	"/subversion{rest:.*}",
	"/whoAmI{rest:.*}",
# my extra urls, to avoid checks
	"/plugin/jquery-ui{rest:.*}",
	"/static/{rest:.*}",
	"/plugin/{rest:.*}",
	"/adjuncts/{rest:.*}",
	"/html5-notifier-plugin/{rest:.*}",
]

[oauth]
client_id = "123098345678-3hg1jdj5nhjr34mnpt473hdb2"
secret = "{your-secret-here}"
callback = "http://your-host-name/oauth2callback"
scope = "https://www.googleapis.com/auth/userinfo.profile+https://www.googleapis.com/auth/userinfo.email"
```

## how it works

Mostly it's just proxy witch adds email (from Google's account) into X-Forwarded-User HTTP Header

Setup reverse proxy in Jenkins:
```
Configure Global Security->Access Control: HTTP Header by reverse proxy
```

Url for manual logout: `/logout`

## TODO:

- versions & dependencies control
- groups support (MAYBE)
