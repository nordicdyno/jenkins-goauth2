package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	_ "expvar"

	"code.google.com/p/goauth2/oauth"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var ProxyHandler *httputil.ReverseProxy

func setupProxyHandler(urlStr string) {
	if *Verbose {
		log.Println("Proxy to:", urlStr)
	}
	u, e := url.Parse(urlStr)
	if e != nil {
		log.Fatal("Bad Proxy url.")
	}
	ProxyHandler = httputil.NewSingleHostReverseProxy(u)
}

var store *sessions.CookieStore

func setupSessionStore(secret string) {
	store = sessions.NewCookieStore([]byte(secret))
}

func main() {
	r := mux.NewRouter()
	u, e := url.Parse(conf.Oauth.Callback)
	if e != nil {
		log.Fatal("Bad callback url")
	}
	if *Verbose {
		log.Println("Register oauth handler on:", u.Path)
	}
	r.HandleFunc(u.Path, oauthHandler)

	r.HandleFunc("/", loginHandler)

	r.HandleFunc("/logout", logoutHandler)

	r.HandleFunc("/dbg{rest:.*}", debugHandler)
	for _, pattern := range conf.Proxy.SkipUrls {
		if *Verbose {
			log.Println("skip url:", pattern)
		}
		r.Handle(pattern, ProxyHandler)
	}
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)

	h := appHandler{router: r}
	os.Args = nil // monkeypatch: hide commandline from expvar
	http.Handle("/", h)

	if *Verbose {
		fmt.Println("Run!")
	}
	err := http.ListenAndServe(conf.Bind, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// handling errors here
type appHandler struct {
	router *mux.Router
}

func (h appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Println("Catch error. Recovering...")
			var doc bytes.Buffer
			err := errorTemplate.Execute(&doc, &ErrorPage{
				Code:    http.StatusInternalServerError,
				Message: rec,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html;charset=utf-8")
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, doc.String())
		}
	}()
	h.router.ServeHTTP(w, r)
}

func notFoundHandler(w http.ResponseWriter, r *http.Request) {
	loginHandler(w, r)
}

func isEmailTrusted(email string) bool {
	if len(conf.TrustedDomain) > 0 {
		if strings.HasSuffix(email, conf.TrustedDomain) {
			return true
		}
	}

	if len(conf.TrustedEmails) > 0 {
		for _, checkEmail := range conf.TrustedEmails {
			if email == checkEmail {
				return true
			}
		}
	}
	return false
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	session.Values["token"] = nil
	session.Values["email"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	email := session.Values["email"]
	emailStr, ok := email.(string)
	if ok && emailStr != "" {
		if !isEmailTrusted(emailStr) {
			panic("Untrusted email: " + emailStr)
		}

		if *Verbose {
			log.Printf("Proxy %s  to %s\n", emailStr, r.URL.String())
		}
		r.Header.Add("X-Forwarded-User", emailStr)

		// "Official" way to avoid url encode
		if conf.Proxy.DisableUrlDecode {
			r.URL.Opaque = strings.SplitN(r.RequestURI, "?", 2)[0]
			if *Verbose {
				log.Println("undecoded url is: ", r.URL.Opaque)
			}
		}

		ProxyHandler.ServeHTTP(w, r)
		return
	}

	var oauthUrl = fmt.Sprintf(oauthLoginUrl,
		conf.Oauth.Callback,
		conf.Oauth.ClientId,
		conf.Oauth.Scope,
		r.RequestURI,
	)

	if *Verbose {
		log.Println("login page render")
	}
	err := loginTemplate.Execute(w, &LoginPage{GoogleUrl: oauthUrl})
	if err != nil {
		panic(err)
	}
}

func debugHandler(w http.ResponseWriter, r *http.Request) {
	dmp, _ := httputil.DumpRequest(r, true)
	fmt.Fprintf(w, string(dmp))
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, sessionName)
	vals := r.URL.Query()
	code := vals.Get("code")

	t := &oauth.Transport{Config: &oauthCfg}
	token, err := t.Exchange(code)
	if err != nil {
		panic(err)
	}

	if *Verbose {
		log.Printf("token: %v", token)
	}

	t.Token = token
	resp, err := t.Client().Get(profileInfoURL)
	defer resp.Body.Close()
	if err != nil {
		panic(err)
	}

	buf, err := ioutil.ReadAll(resp.Body)
	userInfo := UserInfo{}
	err = json.Unmarshal(buf, &userInfo)
	if err != nil {
		panic(err)
	}

	if !isEmailTrusted(userInfo.Email) {
		panic("Untrusted email: " + userInfo.Email)
	}

	// TODO: move to config file/cli option
	session.Options = &sessions.Options{
		Path:     conf.Session.Path,
		MaxAge:   conf.Session.TtlSeconds,
		HttpOnly: conf.Session.HttpOnly,
	}
	session.Values["token"] = token.AccessToken
	session.Values["email"] = userInfo.Email
	session.Save(r, w)

	state := vals.Get("state")
	if len(state) < 1 {
		state = "/"
	}
	http.Redirect(w, r, state, http.StatusFound)
}
