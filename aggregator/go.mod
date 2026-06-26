module github.com/3wayproxy/aggregator

go 1.22

require (
	github.com/3wayproxy/shared v0.0.0
	github.com/gorilla/websocket v1.5.3
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/songgao/water v0.0.0-20200317203138-2b4b6d7c09d8 // indirect
	golang.org/x/sys v0.22.0 // indirect
)

replace github.com/3wayproxy/shared => ../shared
