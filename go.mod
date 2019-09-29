module github.com/quasilyte/phpgrep

go 1.12

require (
	github.com/cznic/golex v0.0.0-20181122101858-9c343928389c // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/z7zmey/php-parser v0.6.0
	gotest.tools v2.2.0+incompatible // indirect
)

replace github.com/z7zmey/php-parser => ./deps/github.com/z7zmey/php-parser
