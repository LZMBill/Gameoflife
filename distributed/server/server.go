package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var curWorld [][]byte
var curTurn int
var mu sync.Mutex

func createNewWorld(height, width int) [][]byte {
	World := make([][]byte, height)
	for i := range World {
		World[i] = make([]byte, width)
	}
	return World
}

func calculateNextState(startY, endY, startX, endX int, world [][]byte) [][]byte {
	ImageHeight := endY - startY
	ImageWidth := endX - startX
	newWorld := createNewWorld(ImageHeight, ImageWidth)
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			neighboursAlive := 0
			for i := -1; i < 2; i++ {
				for j := -1; j < 2; j++ {
					if i == 0 && j == 0 {
						continue
					}
					if world[(y+i+ImageHeight)%ImageHeight][(x+j+ImageWidth)%ImageWidth] != 0 {
						neighboursAlive += 1
					}
				}
			}
			if world[y][x] == 255 {
				if (neighboursAlive < 2) || (neighboursAlive > 3) {
					newWorld[y-startY][x] = 0

				} else {
					newWorld[y-startY][x] = 255
				}
			}
			if world[y][x] == 0 {
				if neighboursAlive == 3 {
					newWorld[y-startY][x] = 255
				} else {
					newWorld[y-startY][x] = 0
				}
			}
		}
	}
	return newWorld
}

type Server struct {
}

func (s *Server) AliveCount(req stubs.Request, res *stubs.Response) (err error) {
	mu.Lock()
	res.World = curWorld
	res.Turn = curTurn
	mu.Unlock()
	return
}

func (s *Server) KeyPress(req stubs.Request, res *stubs.Response) (err error) {
	res.Turn = curTurn
	res.World = curWorld
	return
}

func (s *Server) CloseServer(req stubs.Request, res *stubs.Response) (err error) {
	res.World = curWorld
	os.Exit(0)
	return
}

func (s *Server) Update(req stubs.Request, res *stubs.Response) (err error) {
	curWorld = req.World
	curTurn = 0
	for curTurn < req.Turn {
		mu.Lock()
		curTurn++
		curWorld = calculateNextState(0, req.ImageHeight, 0, req.ImageWidth, curWorld)
		mu.Unlock()
	}
	res.Turn = curTurn
	res.World = curWorld
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Server{})
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("listening on %s", listener.Addr().String())
	defer listener.Close()
	rpc.Accept(listener)
}
