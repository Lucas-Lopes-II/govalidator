module github.com/Lucas-Lopes-II/govalidator/adapters/echo

go 1.25.0

require (
	github.com/Lucas-Lopes-II/govalidator v0.1.0
	github.com/labstack/echo/v4 v4.13.3
)

require (
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/net v0.51.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

// replace is used during local development and CI.
// Remove this directive before tagging adapters/echo/v0.1.0.
replace github.com/Lucas-Lopes-II/govalidator => ../..
