module core

go 1.17

require github.com/apache/thrift v0.16.0

require github.com/vishalmohanty/go_threshenc v0.0.0

replace github.com/vishalmohanty/go_threshenc v0.0.0 => ../crypto/threshenc/go/go_threshenc

require github.com/vishalmohanty/encryption v0.0.0

replace github.com/vishalmohanty/encryption v0.0.0 => ../crypto/threshenc/thrift/gen-go/encryption
