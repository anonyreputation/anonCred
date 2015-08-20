package server
import (
	"net"
	"proto"
	"encoding/gob"
	"bytes"
	"util"
	"fmt"
	"github.com/dedis/crypto/abstract"
	"../github.com/dedis/crypto/shuffle"
	"github.com/dedis/crypto/proof"
	"github.com/dedis/crypto/random"
	"github.com/dedis/crypto/anon"
)

var srcAddr *net.UDPAddr
var anonServer *AnonServer

func Handle(buf []byte,addr *net.UDPAddr, tmpServer *AnonServer, n int) {
	// decode the whole message
	byteArr := make([]util.ByteArray,2)
	gob.Register(byteArr)

	srcAddr = addr
	anonServer = tmpServer
	event := &proto.Event{}
	err := gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(event)
	util.CheckErr(err)
	switch event.EventType {
	case proto.SERVER_REGISTER_REPLY:
		handleServerRegisterReply(event.Params);
		break
	case proto.ANNOUNCEMENT:
		handleAnnouncement(event.Params);
		break
	case proto.UPDATE_NEXT_HOP:
		handleUpdateNextHop(event.Params)
		break
	case proto.CLIENT_REGISTER_SERVERSIDE:
		handleClientRegisterServerSide(event.Params)
		break
	case proto.ROUND_END:
		handleRoundEnd(event.Params)
		break
	default:
		fmt.Println("Unrecognized request")
		break
	}
}

func verifyNeffShuffle(params map[string]interface{}) {
	if _, shuffled := params["shuffled"]; shuffled {
		// get all the necessary parameters
		xbarList := util.ProtobufDecodePointList(params["xbar"].([]byte))
		ybarList := util.ProtobufDecodePointList(params["ybar"].([]byte))
		prevKeyList := util.ProtobufDecodePointList(params["prev_keys"].([]byte))
		prevValList := util.ProtobufDecodePointList(params["prev_vals"].([]byte))
		prePublicKey := anonServer.Suite.Point()
		prePublicKey.UnmarshalBinary(params["public_key"].([]byte))

		// verify the shuffle
		verifier := shuffle.Verifier(anonServer.Suite, nil, prePublicKey, prevKeyList,
			prevValList, xbarList, ybarList)
		err := proof.HashVerify(anonServer.Suite, "PairShuffle", verifier, params["proof"].([]byte))
		if err != nil {
			panic("Shuffle verify failed: " + err.Error())
		}
	}
}

func handleRoundEnd(params map[string]interface{}) {
	keyList := util.ProtobufDecodePointList(params["keys"].([]byte))
	size := len(keyList)
	byteValList := make([][]byte,size)
	if _, ok := params["is_start"]; ok {
		intValList := params["vals"].([]int)
		// The request is sent by coordinator, deserialize the data part
		for i := 0; i < len(intValList); i++ {
			byteValList[i] = util.IntToByte(intValList[i])
		}
	} else {
		// verify neff shuffle if needed
		verifyNeffShuffle(params)
		// deserialize data part
		byteArr := params["vals"].([]util.ByteArray)
		for i := 0; i < len(byteArr); i++ {
			byteValList[i] = byteArr[i].Arr
		}
	}

	rand1 := anonServer.Suite.Cipher([]byte("example"))
	// Create a public/private keypair (X[mine],x)
	X := make([]abstract.Point, 1)
	X[0] = anonServer.PublicKey


	newKeys := make([]abstract.Point,size)
	newVals := make([][]byte,size)
	for i := 0 ; i < size; i++ {
		// decrypt the public key
		newKeys[i] = anonServer.KeyMap[keyList[i].String()]
		// encrypt the reputation using ElGamal algorithm
		C := anon.Encrypt(anonServer.Suite, rand1, byteValList[i], anon.Set(X), false)
		newVals[i] = C
	}
	byteNewKeys := util.ProtobufEncodePointList(newKeys)

	// type is []ByteArr
	byteNewVals := util.SerializeTwoDimensionArray(newVals)

	if(size <= 1) {
		// no need to shuffle, just send the package to next server
		pm := map[string]interface{}{
			"keys" : byteNewKeys,
			"vals" : byteNewVals,
		}
		event := &proto.Event{proto.ROUND_END,pm}
		util.Send(anonServer.Socket,anonServer.PreviousHop,util.Encode(event))
		// reset RoundKey and key map
		anonServer.Roundkey = anonServer.Suite.Secret().Pick(random.Stream)
		anonServer.KeyMap = make(map[string]abstract.Point)
		return
	}

	Xori := make([]abstract.Point, len(newVals))
	for i:=0; i < size; i++ {
		Xori[i] = anonServer.Suite.Point().Mul(nil, anonServer.PrivateKey)
	}
	byteOri := util.ProtobufEncodePointList(Xori)

	rand := anonServer.Suite.Cipher(abstract.RandomKey)
	// *** perform neff shuffle here ***
	Xbar, Ybar, _, Ytmp, prover := neffShuffle(Xori,newKeys,rand)
	prf, err := proof.HashProve(anonServer.Suite, "PairShuffle", rand, prover)
	util.CheckErr(err)


	// this is the shuffled key
	finalKeys := convertToOrigin(Ybar,Ytmp)
	finalVals := rebindReputation(newKeys,newVals,finalKeys)

	// send data to the next server
	byteXbar := util.ProtobufEncodePointList(Xbar)
	byteYbar := util.ProtobufEncodePointList(Ybar)
	byteFinalKeys := util.ProtobufEncodePointList(finalKeys)
	byteFinalVals := util.SerializeTwoDimensionArray(finalVals)
	bytePublicKey, _ := anonServer.PublicKey.MarshalBinary()
	// prev keys means the key before shuffle
	pm := map[string]interface{}{
		"xbar" : byteXbar,
		"ybar" : byteYbar,
		"keys" : byteFinalKeys,
		"vals" : byteFinalVals,
		"proof" : prf,
		"prev_keys": byteOri,
		"prev_vals": byteNewKeys,
		"shuffled":true,
		"public_key" : bytePublicKey,
	}
	event := &proto.Event{proto.ROUND_END,pm}
	util.Send(anonServer.Socket,anonServer.PreviousHop,util.Encode(event))

	// reset RoundKey and key map
	anonServer.Roundkey = anonServer.Suite.Secret().Pick(random.Stream)
	anonServer.KeyMap = make(map[string]abstract.Point)
}

func rebindReputation(newKeys []abstract.Point, newVals [][]byte, finalKeys []abstract.Point) ([][]byte) {
	size := len(newKeys)
	ret := make([][]byte,size)
	m := make(map[string][]byte)
	for i := 0; i < size; i++ {
		m[newKeys[i].String()] = newVals[i]
	}
	for i := 0; i < size; i++ {
		ret[i] = m[finalKeys[i].String()]
	}
	return ret
}

func convertToOrigin(YbarEn, Ytmp []abstract.Point) ([]abstract.Point){
	size := len(YbarEn)
	yyy := make([]abstract.Point, size)

	for i := 0; i < size; i++ {
		yyy[i] = YbarEn[i]
		Ytmp[i].Sub(yyy[i], Ytmp[i])
	}
	return Ytmp
}

// Y is the keys want to shuffle
func neffShuffle(X []abstract.Point, Y []abstract.Point, rand abstract.Cipher) (Xbar, Ybar, Xtmp, Ytmp []abstract.Point, prover proof.Prover){

	Xbar, Ybar, prover, Xtmp, Ytmp = shuffle.Shuffle(anonServer.Suite, nil, anonServer.PublicKey,
		X, Y, rand)
	return
}

// encrypt the public key and send to next hop
func handleClientRegisterServerSide(params map[string]interface{}) {
	publicKey := anonServer.Suite.Point()
	err := publicKey.UnmarshalBinary(params["public_key"].([]byte))
	util.CheckErr(err)

	newKey := anonServer.Suite.Point().Mul(publicKey,anonServer.Roundkey)
	byteNewKey, err := newKey.MarshalBinary()
	util.CheckErr(err)
	pm := map[string]interface{}{
		"public_key" : byteNewKey,
		"addr" : params["addr"].(string),
	}
	event := &proto.Event{proto.CLIENT_REGISTER_SERVERSIDE,pm}
	util.Send(anonServer.Socket,anonServer.NextHop,util.Encode(event))
	// add into key map
	fmt.Println("[debug] Receive client register request... ")
	anonServer.KeyMap[newKey.String()] = publicKey
}

func handleUpdateNextHop(params map[string]interface{}) {
	addr, err := net.ResolveUDPAddr("udp",params["next_hop"].(string))
	util.CheckErr(err)
	anonServer.NextHop = addr
}

func handleAnnouncement(params map[string]interface{}) {
	var g abstract.Point = nil
	keyList := util.ProtobufDecodePointList(params["keys"].([]byte))
	valList := params["vals"].([]util.ByteArray)
	size := len(keyList)

	if val, ok := params["g"]; ok {
		// contains g
		byteG := val.([]byte)
		g = anonServer.Suite.Point()
		g.UnmarshalBinary(byteG)
		g = anonServer.Suite.Point().Mul(g,anonServer.Roundkey)
		// verify the previous shuffle
		verifyNeffShuffle(params)
	}else {
		g = anonServer.Suite.Point().Mul(nil,anonServer.Roundkey)
	}

	X1 := make([]abstract.Point, 1)
	X1[0] = anonServer.PublicKey

	newKeys := make([]abstract.Point,size)
	newVals := make([][]byte,size)
	for i := 0 ; i < len(keyList); i++ {
		// encrypt the public key using modPow
		newKeys[i] = anonServer.Suite.Point().Mul(keyList[i],anonServer.Roundkey)
		// decrypt the reputation using ElGamal algorithm
		MM, err := anon.Decrypt(anonServer.Suite,valList[i].Arr , anon.Set(X1), 0, anonServer.PrivateKey, false)
		util.CheckErr(err)
		newVals[i] = MM
		// update key map
		anonServer.KeyMap[newKeys[i].String()] = keyList[i]
	}
	byteNewKeys := util.ProtobufEncodePointList(newKeys)
	byteNewVals := util.SerializeTwoDimensionArray(newVals)
	byteG, err := g.MarshalBinary()
	util.CheckErr(err)

	if(size <= 1) {
		// no need to shuffle, just send the package to next server
		pm := map[string]interface{}{
			"keys" : byteNewKeys,
			"vals" : byteNewVals,
			"g" : byteG,
		}
		event := &proto.Event{proto.ANNOUNCEMENT,pm}
		util.Send(anonServer.Socket,anonServer.NextHop,util.Encode(event))
		return
	}


	Xori := make([]abstract.Point, len(newVals))
	for i:=0; i < size; i++ {
		Xori[i] = anonServer.Suite.Point().Mul(nil, anonServer.PrivateKey)
	}
	byteOri := util.ProtobufEncodePointList(Xori)

	rand := anonServer.Suite.Cipher(abstract.RandomKey)
	// *** perform neff shuffle here ***
	Xbar, Ybar, _, Ytmp, prover := neffShuffle(Xori,newKeys,rand)
	prf, err := proof.HashProve(anonServer.Suite, "PairShuffle", rand, prover)
	util.CheckErr(err)


	// this is the shuffled key
	finalKeys := convertToOrigin(Ybar,Ytmp)
	finalVals := rebindReputation(newKeys,newVals,finalKeys)

	// send data to the next server
	byteXbar := util.ProtobufEncodePointList(Xbar)
	byteYbar := util.ProtobufEncodePointList(Ybar)
	byteFinalKeys := util.ProtobufEncodePointList(finalKeys)
	byteFinalVals := util.SerializeTwoDimensionArray(finalVals)
	bytePublicKey, _ := anonServer.PublicKey.MarshalBinary()
	// prev keys means the key before shuffle
	pm := map[string]interface{}{
		"xbar" : byteXbar,
		"ybar" : byteYbar,
		"keys" : byteFinalKeys,
		"vals" : byteFinalVals,
		"proof" : prf,
		"prev_keys": byteOri,
		"prev_vals": byteNewKeys,
		"shuffled":true,
		"public_key" : bytePublicKey,
		"g" : byteG,
	}
	event := &proto.Event{proto.ANNOUNCEMENT,pm}
	util.Send(anonServer.Socket,anonServer.NextHop,util.Encode(event))
}

// handle server register reply
func handleServerRegisterReply(params map[string]interface{}) {
	reply := params["reply"].(bool)
	if val, ok := params["prev_server"]; ok {
		ServerAddr, _  := net.ResolveUDPAddr("udp",val.(string))
		anonServer.PreviousHop = ServerAddr
	}
	if reply {
		anonServer.IsConnected = true
	}
}