module github.com/3wayproxy/client

go 1.22

require (
	github.com/3wayproxy/shared v0.0.0
	github.com/gorilla/websocket v1.5.3
	github.com/playwright-community/playwright-go v0.5200.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/deckarep/golang-set/v2 v2.7.0 // indirect
	github.com/go-jose/go-jose/v3 v3.0.4 // indirect
	github.com/go-stack/stack v1.8.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	golang.org/x/sys v0.22.0 // indirect
)

replace github.com/3wayproxy/shared => ../shared
