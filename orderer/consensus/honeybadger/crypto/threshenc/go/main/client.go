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
	"crypto/tls"
	"fmt"
	"os"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/vishalmohanty/encryption"
)

var defaultCtx = context.Background()

type VerificationKey struct {
    key []byte
}

type SecretKey struct {
    key []byte
}

type EncryptedMessage struct {
    U []byte
    V []byte
    W []byte
}

type TPKEPublicKey struct {
    l int32
    k int32
    VK *VerificationKey
    VKs []*VerificationKey
}

type TPKEPrivateKey struct {
    PubKey *TPKEPublicKey
    SK *SecretKey
    i int32
}

type AESKey struct {
    key []byte
}

func getVerificationKeyFromThrift(verKey *encryption.VerificationKeyThrift) (*VerificationKey) {
    return &VerificationKey{
        key: verKey.GetKey(),
    }
}

func convertVerificationKeyToThrift(verKey *VerificationKey) (encryption*VerificationKeyThrift) {
    var verKeyThrift = NewVerificationKeyThrift()
    *verKeyThrift.Key = *verKey.key
    return verKeyThrift
}

func getSecretKeyFromThrift(secKey *encryption.PrivateKeyThrift) (*SecretKey) {
    return &SecretKey{
        key: secKey.GetKey(),
    }
}

func convertSecretKeyToThrift(secretKey *SecretKey) (*encryption.PrivateKeyThrift) {
    var secretKeyThrift = encryption.NewPrivateKeyThrift()
    *secretKeyThrift.Key = *secretKey.key
    return secretKeyThrift
}

func getEncryptedMsgFromThrift(em *encryption.EncryptedMessageThrift) (*EncryptedMessage) {
    return &EncryptedMessage{
        U: em.GetU(),
        V: em.GetV(),
        W: em.GetW()
    }
}

func convertEncryptedMsgToThrift(em *EncryptedMessage) (*encryption.EncryptedMessageThrift) {
    var emThrift = encryption.NewEncryptedMessageThrift()
    *emThrift.U = *em.U
    *emThrift.V = *em.V
    *emThrift.W = *em.W
    return emThrift
}

func getTPKEPublicKeyFromThrift(pubKey *encryption.TPKEPublicKeyThrift) (*TPKEPublicKey) {
    var VKs []*VerificationKey
    for _, vk := range pubKey.GetVKs() {
        VKs = append(VKs, getVerificationKeyFromThrift(vk))  // note the = instead of :=
    }
    return &TPKEPublicKey{
        l: pubKey.GetL(),
        k: pubKey.GetK(),
        VK: getVerificationKeyFromThrift(pubKey.GetVK()),
        VKs: VKs,
    }
}

func convertTPKEPublicKeyToThrift(pubKey *TPKEPublicKey) (*encryption.TPKEPublicKeyThrift) {
    var pubKeyThrift = encryption.NewTPKEPublicKeyThrift()
    *pubKeyThrift.L = *pubKey.l
    *pubKeyThrift.K = *pubKey.k
    *pubKeyThrift.VK = convertVerificationKeyToThrift(*pubKey.VK)

    var VKs []*encryption.VerificationKeyThrift
    for _, vk := range pubKey.VKs {
        VKs = append(VKs, convertVerificationKeyToThrift(vk))  // note the = instead of :=
    }
    *pubKeyThrift.VKs = VKs
    return pubKeyThrift
}

func getTPKEPrivateKeyFromThrift(privKey *encryption.TPKEPrivateKeyThrift) (*TPKEPrivateKey) {
    return &TPKEPrivateKey{
        PubKey: getTPKEPublicKeyFromThrift(privKey.GetPubKey()),
        SK: getSecretKeyFromThrift(privKey.GetSK()),
        i: privKey.GetI(),
    }
}

func convertTPKEPrivateKeyToThrift(privKey *TPKEPrivateKey) (*encryption.TPKEPrivateKeyThrift) {
    var privKeyThrift = encryption.NewTPKEPrivateKeyThrift()
    *privKeyThrift.PubKey = convertTPKEPublicKeyToThrift(*privKey.PubKey)
    *privKeyThrift.SK = convertSecretKeyToThrift(*privKey.SK)
    *privKeyThrift.I = *privKey.I
    return privKeyThrift
}

func getKeysFromDealerThrift(dealerThrift *encryption.DealerThrift) (*TPKEPublicKey, []*TPKEPrivateKey) {
    var privKeys []*TPKEPrivateKey
    for _, pk := range dealerThrift.GetPrivKeys() {
        privKeys = append(privKeys, getTPKEPrivateKeyFromThrift(pk))
    }
    pubKey := getTPKEPublicKeyFromThrift(dealerThrift.GetPubKey())
    return pubKey, privKeys
}

func getAESKeyFromThrift(verKey *encryption.AESKeyThrift) (*AESKey) {
    return &AESKey{
        key: verKey.GetKey(),
    }
}

func convertAESKeyToThrift(verKey *AESKey) (encryption*AESKeyThrift) {
    var aesKeyThrift = NewAESKeyThrift()
    *aesKeyThrift.Key = *aesKey.key
    return aesKeyThrift
}

func getClient() (*encryption.TPKEServiceClient, error) {
    const addr string = "localhost:9090" // Address to listen to
    const secure bool = true // Use tls secure transport
    const buffered bool = false // Use buffered transport
    const framed bool = false // Use framed transport
    const protocol string = "binary" // Specify the protocol (binary, compact, json, simplejson)

    var protocolFactory thrift.TProtocolFactory
	switch protocol {
        case "compact":
            protocolFactory = thrift.NewTCompactProtocolFactoryConf(nil)
        case "simplejson":
            protocolFactory = thrift.NewTSimpleJSONProtocolFactoryConf(nil)
        case "json":
            protocolFactory = thrift.NewTJSONProtocolFactory()
        case "binary", "":
            protocolFactory = thrift.NewTBinaryProtocolFactoryConf(nil)
        default:
            fmt.Fprint(os.Stderr, "Invalid protocol specified", protocol, "\n")
            Usage()
            os.Exit(1)
	}

	var transportFactory thrift.TTransportFactory
	cfg := &thrift.TConfiguration{
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	if buffered {
		transportFactory = thrift.NewTBufferedTransportFactory(8192)
	} else {
		transportFactory = thrift.NewTTransportFactory()
	}

	if framed {
		transportFactory = thrift.NewTFramedTransportFactoryConf(transportFactory, cfg)
	}

	var transport thrift.TTransport
	if secure {
		transport = thrift.NewTSSLSocketConf(addr, cfg)
	} else {
		transport = thrift.NewTSocketConf(addr, cfg)
	}
	transport, err := transportFactory.GetTransport(transport)
	if err != nil {
		return nil, err
	}
	defer transport.Close()
	if err := transport.Open(); err != nil {
		return nil, err
	}
	iprot := protocolFactory.GetProtocol(transport)
	oprot := protocolFactory.GetProtocol(transport)

	return encryption.NewTPKEServiceClient(thrift.NewTStandardClient(iprot, oprot)), nil
}

func (pubKey *TPKEPublicKey) lagrange(S []int32, j int32) int32 {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    pubKeyThrift := convertTPKEPublicKeyToThrift(pubKey)

    val, err := client.Lagrange(defaultCtx, pubKeyThrift, S, j)
    if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", val)
	}

	return val
}

func (pubKey *TPKEPublicKey) encrypt(m string) (U []byte, V []byte, W []byte) {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    pubKeyThrift := convertTPKEPublicKeyToThrift(pubKey)

    val, err := client.Encrypt(defaultCtx, pubKeyThrift, m)
    if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", val)
	}

	return val.GetU(), val.GetV(), val.GetW()
}

func (pubKey *TPKEPublicKey) combineShares(U []byte, V []byte, W []byte, shares map[int32][]byte) []byte {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    pubKeyThrift := convertTPKEPublicKeyToThrift(pubKey)
    var emThrift = encryption.NewEncryptedMessageThrift()
    *emThrift.U = U
    *emThrift.V = V
    *emThrift.W = W

    val, err := client.CombineShares(defaultCtx, pubKeyThrift, emThrift, shares)
    if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", val)
	}

	return val
}

func (privKey *TPKEPrivateKey) decryptShare(U []byte, V []byte, W []byte) []byte {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    privKeyThrift := convertTPKEPrivateKeyToThrift(privKey)
    var emThrift = encryption.NewEncryptedMessageThrift()
    *emThrift.U = U
    *emThrift.V = V
    *emThrift.W = W

    val, err := client.DecryptShare(defaultCtx, privKeyThrift, emThrift)
    if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", val)
	}

	return val
}

func dealer(players int32, k int32) (*TPKEPublicKey, []*TPKEPrivateKey, error) {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    var pubKey *TPKEPublicKey
    var privKeys []*TPKEPrivateKey
    dealerThrift, err := client.Dealer(defaultCtx, players, k)

	if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", dealerThrift)
		pubKey, privKeys = getKeysFromDealerThrift(dealerThrift)
	}

	return pubKey, privKeys, err
}

func (aesKey *AESKey) aesEncrypt(raw []byte) []byte {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    aesKeyThrift := convertAESKeyToThrift(aesKey)

    val, err := client.AesEncrypt(defaultCtx, aesKeyThrift, raw)
    if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", val)
	}

	return val
}

func (aesKey *AESKey) aesDecrypt(encMes []byte) []byte {
    var client *encryption.TPKEServiceClient
    client, err := getClient()

    if err != nil {
        fmt.Println("Could not initialize client due to ", err)
        os.Exit(1)
    }

    aesKeyThrift := convertAESKeyToThrift(aesKey)

    val, err := client.AesDecrypt(defaultCtx, aesKeyThrift, encMes)
    if err != nil {
        fmt.Println("Error during operation: ", err)
	} else {
		fmt.Println("Success! ", val)
	}

	return val
}

func handleClient(client *encryption.TPKEServiceClient) (err error) {
    ver_key := encryption.NewVerificationKeyThrift()
    ver_key.Key = []byte{71, 111}
    pub_key := encryption.NewTPKEPublicKeyThrift()
    pub_key.L = 10
    pub_key.K = 5
    pub_key.VK = ver_key
    pub_key.VKs = []*encryption.VerificationKeyThrift{ver_key}

    dealerThrift, err := client.Dealer(defaultCtx, 10, 5)

	if err != nil {
        fmt.Println("Error during operation:", err)
	} else {
		fmt.Println("Success! ", dealerThrift)
		pubKey, privKeys := getKeysFromDealerThrift(dealerThrift)
		fmt.Println("Public key: ", *pubKey)
		fmt.Println("Private keys: ", privKeys)
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