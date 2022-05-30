module core

go 1.17

replace github.com/vishalmohanty/go_threshenc v0.0.0 => ../crypto/threshenc/go/go_threshenc

require threshsig v0.0.0

replace threshsig v0.0.0 => ../crypto/threshsig

require (
	github.com/hpcloud/tail v1.0.0
	github.com/juju/fslock v0.0.0-20160525022230-4d5c94c67b4b
	github.com/klauspost/reedsolomon v1.9.16
	github.com/niclabs/tcrsa v0.0.5
	github.com/pebbe/zmq4 v1.2.8
)

require (
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/klauspost/cpuid/v2 v2.0.6 // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 // indirect
	gopkg.in/fsnotify.v1 v1.4.7 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
)

replace github.com/vishalmohanty/encryption v0.0.0 => ../crypto/threshenc/thrift/gen-go/encryption
