package main

/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements. See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership. The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License. You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

import (
	"context"
	"fmt"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/vishalmohanty/encryption"
)

var defaultCtx = context.Background()

func handleClient(client *encryption.TPKEServiceClient) (err error) {
// 	client.Ping(defaultCtx)
// 	fmt.Println("ping()")

    ver_key := encryption.NewVerificationKeyThrift()
    ver_key.Key = []byte{71, 111}
    pub_key := encryption.NewTPKEPublicKeyThrift()
    pub_key.L = 10
    pub_key.K = 5
    pub_key.VK = ver_key
    pub_key.VKs = []*encryption.VerificationKeyThrift{ver_key}

// 	val, err := client.Lagrange(
// 	    defaultCtx,
// 	    pub_key,
// 	    []int32{0, 1},
// 	    0,
// 	)
    val, err := client.Dealer(defaultCtx, 10, 5)

	if err != nil {
// 		switch v := err.(type) {
// 		case *tutorial.InvalidOperation:
// 			fmt.Println("Invalid operation:", v)
// 		default:
        fmt.Println("Error during operation:", err)
// 		}
	} else {
		fmt.Println("Success! ", val)
	}
	return err
}

func runClient(transportFactory thrift.TTransportFactory, protocolFactory thrift.TProtocolFactory, addr string, secure bool, cfg *thrift.TConfiguration) error {
	var transport thrift.TTransport
	if secure {
		transport = thrift.NewTSSLSocketConf(addr, cfg)
	} else {
		transport = thrift.NewTSocketConf(addr, cfg)
	}
	transport, err := transportFactory.GetTransport(transport)
	if err != nil {
		return err
	}
	defer transport.Close()
	if err := transport.Open(); err != nil {
		return err
	}
	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)
	return handleClient(encryption.NewTPKEServiceClient(thrift.NewTStandardClient(iprot, oprot)))
}