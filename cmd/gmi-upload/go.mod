module scsy.de/gmi-upload

go 1.19

require (
	github.com/akamensky/argparse v1.4.0
	github.com/go-resty/resty/v2 v2.7.0
	github.com/pkg/xattr v0.4.9
	github.com/tidwall/gjson v1.14.4
	verboseOutput v0.0.0-00010101000000-000000000000
)

require (
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.0 // indirect
	golang.org/x/net v0.0.0-20211029224645-99673261e6eb // indirect
	golang.org/x/sys v0.0.0-20220408201424-a24fb2fb8a0f // indirect
)

replace verboseOutput => ../../internal/verboseOutput
