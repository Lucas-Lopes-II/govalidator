module github.com/Lucas-Lopes-II/govalidator/adapters/fiber

go 1.25.0

require (
	github.com/Lucas-Lopes-II/govalidator v0.1.0
	github.com/gofiber/fiber/v2 v2.52.5
)

require (
	github.com/andybalholm/brotli v1.0.5 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.51.0 // indirect
	github.com/valyala/tcplisten v1.0.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
)

// replace is used during local development and CI.
// Remove this directive before tagging adapters/fiber/v0.1.0.
replace github.com/Lucas-Lopes-II/govalidator => ../..
