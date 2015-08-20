package main
import (


	"fmt"
	"util"

	"proto"
	"encoding/gob"
	"bytes"
)
type Message struct {
	Nym     []byte
}


func main() {
	/*
	suite1 := nist.NewAES128SHA256QR512()
	l := make([]abstract.Point,1)
	key := suite1.Secret().Pick(random.Stream)
	l[0] = suite1.Point().Mul(nil,key)
	m := make(map[string]int)
	m[l[0].String()] = 2
	bytes := util.ProtobufEncodePointList(l)
	keyList := util.ProtobufDecodePointList(bytes)

	fmt.Println(m[l[0].String()])
	fmt.Println(m[keyList[0].String()])
	fmt.Println(l[0].String())
	fmt.Println(keyList[0].String())
	*/
	/*
	suite1 := nist.NewAES128SHA256QR512()
	key := suite1.Secret().Pick(random.Stream)
	publicKey := suite1.Point().Mul(nil,key)
	var a int = 1
	aBytes := util.IntToByte(a)
	_,_,data := util.ElGamalEncrypt(suite1,publicKey,aBytes)
	fmt.Println(data)
	_,_,qdata := util.ElGamalEncrypt(suite1,publicKey,data)
	fmt.Println(qdata)
	*/
	var a int = 1
	aBytes := util.IntToByte(a)
	arr := make([]Message,2)
	gob.Register(arr)
	arr[0].Nym = aBytes
	arr[1].Nym = aBytes
	pm := map[string]interface{}{
		"keys" : arr,
	}
	event := &proto.Event{proto.ROUND_END,pm}
	data := util.Encode(event)
	newEvent := &proto.Event{}
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(newEvent)
	util.CheckErr(err)
	fmt.Println((newEvent.Params["keys"].([]Message))[0])

	/*
	suite := nist.NewAES128SHA256QR512()
	key := suite.Secret().Pick(random.Stream)
	publicKey := suite.Point().Mul(nil,key)
	var a int = 0
	s := make([]abstract.Point,2)
	s[0] = nist.NewResidueGroup().NewPoint(int64(a))
	fmt.Println(s[0].String())

	K,C,_ := util.ElGamalEncrypt(suite,publicKey,s[0])
	K1,C1,_ := util.ElGamalEncrypt(suite,publicKey,C)
	P1 := util.ElGamalDecrypt(suite, key, K1, C1)
	P := util.ElGamalDecrypt(suite, key, K, P1)
	//_,_,qdata := util.ElGamalEncrypt(suite,publicKey,data)
	fmt.Println(P)
	*/
	/*
	suite := nist.NewAES128SHA256QR512()
	rand := suite.Cipher([]byte("example"))

	// Create a public/private keypair (X[mine],x)
	X := make([]abstract.Point, 1)
	mine := 0                           // which public key is mine
	x := suite.Secret().Pick(rand)      // create a private key x
	X[mine] = suite.Point().Mul(nil, x) // corresponding public key X

	// Encrypt a message with the public key
	var a int = 3
	M := util.IntToByte(a)
	C := anon.Encrypt(suite, rand, M, anon.Set(X), false)
	//fmt.Printf("Encryption of '%s':\n%s", string(M), hex.Dump(C))
	//fmt.Println(string(C))

	// another
	rand = suite.Cipher([]byte("example"))
	X1 := make([]abstract.Point, 1)
	x1 := suite.Secret().Pick(rand)      // create a private key x
	X1[0] = suite.Point().Mul(nil, x1) // corresponding public key X
	C1 := anon.Encrypt(suite, rand, C, anon.Set(X1), false)

	p,pdw := suite.Point().Pick(C1,random.)
	suite.Point().
	fmt.Println(C1)
	pd, _ := p.Data()
	fmt.Println(pd)
	fmt.Println(pdw)
suite.
	// Decrypt the ciphertext with the private key
	MM, err := anon.Decrypt(suite,pd , anon.Set(X1), 0, x1, false)
	if err != nil {
		panic(err.Error())
	}
	MF, err1 := anon.Decrypt(suite, MM, anon.Set(X), 0, x, false)
	if err1 != nil {
		panic(err.Error())
	}

	fmt.Println(util.ByteToInt(MF))
	*/

	/*
	suite1 := nist.NewAES128SHA256QR512()
	suite2 := nist.NewAES128SHA256QR512()
	suite3 := nist.NewAES128SHA256QR512()
	key := suite1.Secret().Pick(random.Stream)
	g1 := suite1.Point().Mul(nil,key)
	byte1, _ := g1.MarshalBinary()
	fmt.Println(g1)

	g2 := suite2.Point()
	err := g2.UnmarshalBinary(byte1)
	g2 = suite2.Point().Mul(g2,key)
	bytes2, _ := g2.MarshalBinary()
	fmt.Println(g2)

	g3 := suite3.Point()
	err = g3.UnmarshalBinary(bytes2)
	util.CheckErr(err)
	g3 = suite3.Point().Mul(g3,key)

	fmt.Println(g3)
	*/

	/*
	var aSecret abstract.Secret
	var tSecret = reflect.TypeOf(&aSecret).Elem()

	suite := nist.NewAES128SHA256QR512()
	cons := protobuf.Constructors {
		tSecret: func()interface{} { return suite.Secret() },
	}

	a := suite.Secret().Pick(random.Stream)
	b := suite.Secret().Pick(random.Stream)
	fmt.Println(a)
	fmt.Println(b)

	byteA, _ := a.MarshalBinary()
	byteB, _ := b.MarshalBinary()
	l := map[string][]byte {
		"a":byteA,
		"b":byteB,
	}

	byteNym, err := protobuf.Encode(&Message{l})
	if err != nil {
		fmt.Println(err.Error())
	}

	var msg Message
	if err = protobuf.DecodeWithConstructors(byteNym, &msg, cons); err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(msg.Nym["a"])
	fmt.Println(msg.Nym["b"])
	*/
}
