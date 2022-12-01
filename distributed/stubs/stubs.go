package stubs

//Server

var Gameoflife = "Server.Update"
var KeyPress = "Server.KeyPress"
var AliveCount = "Server.AliveCount"
var CloseServer = "Server.CloseServer"

type Request struct {
	World       [][]byte
	Turn        int
	ImageHeight int
	ImageWidth  int
	Threads     int
}

type Response struct {
	World [][]byte
	Turn  int
}
