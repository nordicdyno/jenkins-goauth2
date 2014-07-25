package main

import (
	"html/template"
)

var userInfoTemplate = template.Must(template.New("").Parse(`
<html><body>
This app is now authenticated to access your Google user info.  Your details are:<br />
{{.}}
</body></html>
`))

type ErrorPage struct {
	Code    int
	Message interface{}
}

var errorTemplate = template.Must(template.New("").Parse(`
<html><body>
<h2>This app is crashed with error:</h2>
<h2>Code: {{.Code}}<br>
Message: «{{.Message}}»
</h2>
<a href="/">return to main page</a>
</body></html>
`))

type LoginPage struct {
	GoogleUrl string
	Admin     string
}

var loginTemplate = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
<head><title>Login page</title></head>
<body>
<div style="text-align: center; font-size: 80%; font-family: Arial, sans-serif">
    <p><a href="{{.GoogleUrl}}">Log in</a> with your Google account</p>
    <p style="margin-top: 3em">
    For access, contact with <a href="{{.Admin}}">administrator</a>.
    </p>
:D
</div>
</body>
</html>
`))
