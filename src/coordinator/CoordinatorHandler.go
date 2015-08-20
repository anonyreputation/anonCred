package coordinator


import (
	"net"
	"encoding/gob"
	"../proto"
	"fmt"

	"bytes"
	"../util"
	"strings"
	"strconv"
	"time"
	"github.com/dedis/crypto/abstract"
)

var anonCoordinator *Coordinator
var srcAddr *net.UDPAddr

func Handle(buf []byte,addr *net.UDPAddr, tmpCoordinator *Coordinator, n int) {
	// decode the whole message
	anonCoordinator = tmpCoordinator
	srcAddr = addr

	event := &proto.Event{}
	err := gob.NewDecoder(bytes.NewReader(buf[:n])).Decode(event)
	util.CheckErr(err)
	switch event.EventType {
	case proto.SERVER_REGISTER:
		handleServerRegister()
		break
	case proto.CLIENT_REGISTER_CONTROLLERSIDE:
		handleClientRegisterControllerSide(event.Params,);
		break
	case proto.CLIENT_REGISTER_SERVERSIDE:
		handleClientRegisterServerSide(event.Params);
		break
	case proto.MESSAGE:
		handleMsg(event.Params)
		break
	case proto.VOTE:
		handleVote(event.Params)
		break
	case proto.ROUND_END:
		handleRoundEnd(event.Params)
		break
	case proto.ANNOUNCEMENT:
		handleAnnouncement(event.Params)
		break
	default:
		fmt.Println("[fatal] Unrecognized request...")
		break
	}
}


// Handler for ANNOUNCEMENT event
// finish announcement and send start message signal to the clients
func handleAnnouncement(params map[string]interface{}) {
	// This event is triggered when server finishes announcement
	// distribute final reputation map to servers
	if len(params["keys"].([]byte)) == 0 {
		// suggest there is no client
		anonCoordinator.Status = MESSAGE
		return
	}
	var g = anonCoordinator.Suite.Point()
	byteG := params["g"].([]byte)
	err := g.UnmarshalBinary(byteG)
	util.CheckErr(err)

	//construct Decrypted reputation map
	keyList := util.ProtobufDecodePointList(params["keys"].([]byte))
	valList := params["vals"].([]util.ByteArray)
	anonCoordinator.DecryptedReputationMap = make(map[string]int)
	anonCoordinator.DecryptedKeysMap = make(map[string]abstract.Point)

	for i := 0; i < len(keyList); i++ {
		val := util.ByteToInt(valList[i].Arr)
		anonCoordinator.AddIntoDecryptedMap(keyList[i],val)
	}

	// distribute g and hash table of ids to user
	pm := map[string]interface{}{
		"g": params["g"].([]byte),
	}

	event := &proto.Event{proto.ANNOUNCEMENT,pm}
	for _,val := range anonCoordinator.Clients {
		util.Send(anonCoordinator.Socket,val,util.Encode(event))
	}

	// set controller's new g
	anonCoordinator.G = g
	anonCoordinator.Status = MESSAGE
}

// handle server register request
func handleServerRegister() {
	fmt.Println("[debug] Receive the registration info from server " + srcAddr.String());
	// send reply to the new server
	lastServer := anonCoordinator.GetLastServer()

	// update next hop for previous server
	if lastServer != nil {
		pm2 := map[string]interface{}{
			"reply": true,
			"next_hop": srcAddr.String(),
		}
		event2 := &proto.Event{proto.UPDATE_NEXT_HOP, pm2}
		util.Send(anonCoordinator.Socket, lastServer, util.Encode(event2))
	}

	if lastServer == nil {
		lastServer = anonCoordinator.LocalAddr
	}
	pm1 := map[string]interface{}{
		"reply": true,
		"prev_server": lastServer.String(),
	}
	event1 := &proto.Event{proto.SERVER_REGISTER_REPLY,pm1}
	util.Send(anonCoordinator.Socket,srcAddr,util.Encode(event1))

	anonCoordinator.AddServer(srcAddr);
}

// Handler for REGISTER event
// send the register request to server to do encryption
func handleClientRegisterControllerSide(params map[string]interface{}) {
	// get client's public key
	publicKey := anonCoordinator.Suite.Point()
	publicKey.UnmarshalBinary(params["public_key"].([]byte))
	anonCoordinator.AddClient(publicKey,srcAddr)

	// send register info to the first server
	firstServer := anonCoordinator.GetFirstServer()
	pm := map[string]interface{}{
		"public_key": params["public_key"],
		"addr": srcAddr.String(),
	}
	event := &proto.Event{proto.CLIENT_REGISTER_SERVERSIDE,pm}
	util.Send(anonCoordinator.Socket,firstServer,util.Encode(event))
}

// handle client register successful event
func handleClientRegisterServerSide(params map[string]interface{}) {
	// get public key from params (it's one-time nym actually)
	var publicKey = anonCoordinator.Suite.Point()
	bytePublicKey := params["public_key"].([]byte)
	publicKey.UnmarshalBinary(bytePublicKey)

	var addrStr = params["addr"].(string)
	addr,err := net.ResolveUDPAddr("udp",addrStr)
	util.CheckErr(err)
	pm := map[string]interface{}{}
	event := &proto.Event{proto.CLIENT_REGISTER_CONFIRMATION,pm}
	util.Send(anonCoordinator.Socket,addr,util.Encode(event))

	// instead of sending new client to server, we will send it when finishing this round. Currently we just add it into buffer
	anonCoordinator.AddClientInBuffer(publicKey)
}

// verify the msg and broadcast to clients
func handleMsg(params map[string]interface{}) {
	// get info from the request
	text := params["text"].(string)
	byteSig := params["signature"].([]byte)
	nym := anonCoordinator.Suite.Point()
	byteNym := params["nym"].([]byte)
	err := nym.UnmarshalBinary(byteNym)
	util.CheckErr(err)

	fmt.Println("[debug] Receiving msg from " + srcAddr.String() + ": " + text)
	// verify the identification of the client

	byteText := []byte(text)
	err = util.ElGamalVerify(anonCoordinator.Suite,byteText,nym,byteSig,anonCoordinator.G)
	if err != nil {
		fmt.Print("[note]** Fails to verify the message...")
		return
	}
	// add msg log
	msgID := anonCoordinator.AddMsgLog(nym)

	// generate msg to clients
	pm := map[string]interface{}{
		"text" : text,
		"nym" : params["nym"].([]byte),
		"rep" : anonCoordinator.GetReputation(nym),
		"msgID" : msgID,
	}
	event := &proto.Event{proto.MESSAGE,pm}

	// send to all the clients
	for _,val := range anonCoordinator.Clients {
		util.Send(anonCoordinator.Socket,val,util.Encode(event))
	}
	// send confirmation to msg sender
	pm_msg := map[string]interface{}{
		"reply" : true,
	}
	event1 := &proto.Event{proto.MSG_REPLY,pm_msg}
	util.Send(anonCoordinator.Socket,srcAddr,util.Encode(event1))
}

// verify the vote and reply to client
func handleVote(params map[string]interface{}) {
	// get info from the request
	text := params["text"].(string)
	byteSig := params["signature"].([]byte)
	nym := anonCoordinator.Suite.Point()
	byteNym := params["nym"].([]byte)
	err := nym.UnmarshalBinary(byteNym)
	util.CheckErr(err)

	fmt.Println("[debug] Receiving vote from " + srcAddr.String() + ": " + text)
	// verify the identification of the client

	byteText := []byte(text)
	err = util.ElGamalVerify(anonCoordinator.Suite,byteText,nym,byteSig, anonCoordinator.G)
	var pm map[string]interface{}
	if err != nil {
		fmt.Print("[note]** Fails to verify the vote...")
		pm = map[string]interface{}{
			"reply" : false,
		}
		return
	}else {
		// avoid duplicate vote
		// todo

		// get msg id and vote
		commands := strings.Split(text,";")
		// modify the reputation
		msgID, _ := strconv.Atoi(commands[0])
		vote, _ := strconv.Atoi(commands[1])
		targetNym := anonCoordinator.MsgLog[msgID-1]

		anonCoordinator.DecryptedReputationMap[targetNym.String()] =
					anonCoordinator.DecryptedReputationMap[targetNym.String()] + vote
		// generate reply msg to client
		pm = map[string]interface{}{
			"reply" : true,
		}
	}

	event := &proto.Event{proto.VOTE_REPLY,pm}
	// send reply to the client
	util.Send(anonCoordinator.Socket,srcAddr,util.Encode(event))
}

// Handler for ROUND_END event
// send user round end notification
func handleRoundEnd(params map[string]interface{}) {
	// review reputation map
	keyList := util.ProtobufDecodePointList(params["keys"].([]byte))
	valList := params["vals"].([]util.ByteArray)
	anonCoordinator.ReputationMap = make(map[string][]byte)
	anonCoordinator.ReputationKeyMap = make(map[string]abstract.Point)
	for i := 0; i < len(keyList); i++ {
		anonCoordinator.ReputationMap[keyList[i].String()] = valList[i].Arr
		anonCoordinator.ReputationKeyMap[keyList[i].String()] = keyList[i]
	}

	// send user round-end message
	pm := map[string]interface{} {}
	event := &proto.Event{proto.ROUND_END,pm}
	for _,val := range anonCoordinator.Clients {
		util.Send(anonCoordinator.Socket,val,util.Encode(event))
	}
	time.Sleep(500 * time.Millisecond)
	anonCoordinator.Status = READY_FOR_NEW_ROUND
}

