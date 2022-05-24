module core

go 1.17

require github.com/apache/thrift v0.16.0

require github.com/vishalmohanty/go_threshenc v0.0.0

replace github.com/vishalmohanty/go_threshenc v0.0.0 => ../crypto/threshenc/go/go_threshenc

require (
	github.com/cbergoon/merkletree v0.2.0
	github.com/deckarep/golang-set v1.8.0
	github.com/klauspost/reedsolomon v1.9.16
	github.com/niclabs/tcrsa v0.0.5
	github.com/pebbe/zmq4 v1.2.8
	github.com/stretchr/testify v1.7.1
	github.com/vishalmohanty/encryption v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.0 // indirect
	github.com/klauspost/cpuid/v2 v2.0.6 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/vishalmohanty/encryption v0.0.0 => ../crypto/threshenc/thrift/gen-go/encryption
