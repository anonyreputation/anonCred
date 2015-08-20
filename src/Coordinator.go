package main

import (

	"net"
	"fmt"
	"./util"
	"./coordinator"
	"bufio"
	"os"
	"github.com/dedis/crypto/nist"
	"github.com/dedis/crypto/random"
	"time"
	"log"
	"github.com/dedis/crypto/abstract"
	"proto"
)

// pointer to coordinator itself
var anonCoordinator *coordinator.Coordinator

/**
  * start server listener to handle event
  */
func startServerListener() {
	fmt.Println("[debug] Coordinator server listener started...");
	buf := make([]byte, 4096)
	for {
		n,addr,err := anonCoordinator.Socket.ReadFromUDP(buf)
		util.CheckErr(err)
		coordinator.Handle(buf,addr,anonCoordinator,n)
	}
}

/**
  * initialize coordinator
  */
func initCoordinator() {
	config := util.ReadConfig()
	fmt.Println(config)
	ServerAddr,err := net.ResolveUDPAddr("udp","127.0.0.1:"+config["local_port"])
	util.CheckErr(err)
	suite := nist.NewAES128SHA256QR512()
	a := suite.Secret().Pick(random.Stream)
	A := suite.Point().Mul(nil, a)

	anonCoordinator = &coordinator.Coordinator{ServerAddr,nil,nil,
		coordinator.CONFIGURATION,suite,a,A,nil, make(map[string]*net.UDPAddr),
		make(map[string]abstract.Point), make(map[string][]byte), nil, nil,
		make(map[string]int),make(map[string]abstract.Point)}
}


/**
 * clear all buffer data
 */
func clearBuffer() {
	// clear buffer
	anonCoordinator.NewClientsBuffer = nil
	// msg sender's record nym
	anonCoordinator.MsgLog = nil
}

/**
  * send announcement signal to first server
  * send reputation list
  */
func announce() {
	firstServer := anonCoordinator.GetFirstServer()
	if firstServer == nil {
		anonCoordinator.Status = coordinator.MESSAGE
		return
	}
	// construct reputation list (public & encrypted reputation)
	size := len(anonCoordinator.ReputationMap)
	keys := make([]abstract.Point,size)
	vals := make([][]byte,size)
	i := 0
	for k, v := range anonCoordinator.ReputationMap {
		keys[i] = anonCoordinator.ReputationKeyMap[k]
		vals[i] = v
		i++
	}
	byteKeys := util.ProtobufEncodePointList(keys)
	byteVals := util.SerializeTwoDimensionArray(vals)
	params := map[string]interface{}{
		"keys" : byteKeys,
		"vals" : byteVals,
	}
	event := &proto.Event{proto.ANNOUNCEMENT,params}
	util.Send(anonCoordinator.Socket,firstServer,util.Encode(event))
}

/**
 * send round-end signal to last server in topology
 * add new clients into the reputation map
 */
func roundEnd() {
	lastServer := anonCoordinator.GetLastServer()
	if lastServer == nil {
		anonCoordinator.Status = coordinator.READY_FOR_NEW_ROUND
		return
	}
	// add new clients into reputation map
	for _,nym := range anonCoordinator.NewClientsBuffer {
		anonCoordinator.AddIntoDecryptedMap(nym,0)
	}
	// add previous clients into reputation map
	// construct the parameters
	size := len(anonCoordinator.DecryptedReputationMap)
	keys := make([]abstract.Point,size)
	vals := make([]int,size)
	i := 0
	for k, v := range anonCoordinator.DecryptedReputationMap {
		keys[i] = anonCoordinator.DecryptedKeysMap[k]
		vals[i] = v
		i++
	}
	byteKeys := util.ProtobufEncodePointList(keys)
	// send signal to server
	pm := map[string]interface{} {
		"keys" : byteKeys,
		"vals" : vals,
		"is_start" : true,
	}
	event := &proto.Event{proto.ROUND_END,pm}
	util.Send(anonCoordinator.Socket,lastServer,util.Encode(event))

}

/**
 * start vote phase, actually, if we partition the clients to servers,
 * we can let server send this signal to clients. Here, for simplicity, we
 * just send it from controller
 */
func vote() {
	pm := map[string]interface{} {}
	event := &proto.Event{proto.VOTE,pm}
	for _, val :=  range anonCoordinator.Clients {
		util.Send(anonCoordinator.Socket, val, util.Encode(event))
	}
}


func main() {
	// init coordinator
	initCoordinator()
	// bind to socket
	conn, err := net.ListenUDP("udp",anonCoordinator.LocalAddr )
	util.CheckErr(err)
	anonCoordinator.Socket = conn
	// start listener
	go startServerListener()
	fmt.Println("** Note: Type ok to finish the server configuration. **")
	// read ok to start life cycle
	reader := bufio.NewReader(os.Stdin)
	for {
		data, _, _ := reader.ReadLine()
		command := string(data)
		if command == "ok" {
			break
		}
	}
	fmt.Println("[debug] Servers in the current network:")
	fmt.Println(anonCoordinator.ServerList)
	anonCoordinator.Status = coordinator.READY_FOR_NEW_ROUND
	for {
		// wait for the status changed to READY_FOR_NEW_ROUND
		for i := 0; i < 100; i++ {
			if anonCoordinator.Status == coordinator.READY_FOR_NEW_ROUND {
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
		// clear buffer at the beginning of each round
		clearBuffer()
		fmt.Println("******************** New round begin ********************")
		if anonCoordinator.Status != coordinator.READY_FOR_NEW_ROUND {
			log.Fatal("Fails to be ready for the new round")
			os.Exit(1)
		}
		anonCoordinator.Status = coordinator.ANNOUNCE
		fmt.Println("[coordinator] Announcement phase started...")
		// start announce phase
		announce()
		for i := 0; i < 100; i++ {
			if anonCoordinator.Status == coordinator.MESSAGE {
				break
			}
			time.Sleep(1000 * time.Millisecond)
		}
		if anonCoordinator.Status != coordinator.MESSAGE {
			log.Fatal("Fails to be ready for message phase")
			os.Exit(1)
		}
		// start message and vote phase
		fmt.Println("[coordinator] Messaging phase started...")
		// 10 secs for msg
		time.Sleep(10000 * time.Millisecond)
		vote()
		fmt.Println("[coordinator] Voting phase started...")
		// 10 secs for vote
		time.Sleep(10000 * time.Millisecond)
		roundEnd()
	}
}