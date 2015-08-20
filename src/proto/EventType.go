package proto



// server register event
const SERVER_REGISTER = 1;
// reply to server register request
const SERVER_REGISTER_REPLY = 2
// update next hop for server
const UPDATE_NEXT_HOP = 3
// register client request to controller
const CLIENT_REGISTER_CONTROLLERSIDE = 4
// register client request to server
const CLIENT_REGISTER_SERVERSIDE = 5
// confirmation for successfully registering client
const CLIENT_REGISTER_CONFIRMATION = 6
// add a new client
const ADD_NEWCLIENT = 7
// announce phase event
const ANNOUNCEMENT = 8
// synchronize reputation map among servers
const SYNC_REPMAP = 9
// message phase event
const MESSAGE = 10
// vote phase event
const VOTE = 11
// round end event
const ROUND_END = 12
// return vote status event
const VOTE_REPLY = 13
// return msg status event
const MSG_REPLY = 14

