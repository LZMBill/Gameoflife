package gol

import (
	"fmt"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

func load(p Params, c distributorChannels) [][]byte {
	c.ioCommand <- ioInput
	World := createNewWorld(p.ImageHeight, p.ImageWidth)
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			input := <-c.ioInput
			if input != 0 {
				World[y][x] = input
			}
		}
	}
	return World
}

func distributor(p Params, c distributorChannels) {
	World := createNewWorld(p.ImageHeight, p.ImageWidth)
	client, err := rpc.Dial("tcp", "127.0.0.1:8030")
	if err != nil {
		return
	}
	defer client.Close()
	World = load(p, c)
	//ticker
	ticker := time.NewTicker(time.Second * 2)
	go func() {
		for {
			select {
			case <-ticker.C:
				var res stubs.Response
				var req stubs.Request
				err := client.Call(stubs.AliveCount, req, &res)
				if err != nil {
					fmt.Println("alive cells count:", err)
					Exit(0, c)
					close(c.events)
				}
				c.events <- AliveCellsCount{res.Turn, len(calculateAliveCells(p, res.World))}
			case key := <-c.keyPresses:
				var res stubs.Response
				var req stubs.Request
				err := client.Call(stubs.KeyPress, req, &res)
				if err != nil {
					fmt.Println("error:", err)
				}
				switch key {
				case 'q':
					Exit(res.Turn, c)
					close(c.events)
					os.Exit(0)
				case 's':
					filename := fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, res.Turn)
					output(p, c, res.World, filename)
					c.events <- ImageOutputComplete{res.Turn, filename}
				case 'k':
					World = res.World
					filename := fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, res.Turn)
					output(p, c, World, filename)
					c.events <- ImageOutputComplete{res.Turn, filename}
					err := client.Call(stubs.CloseServer, stubs.Request{}, stubs.Response{})
					if err != nil {
						fmt.Println("error:", err)
					}
					if key == 'k' {
						Exit(res.Turn, c)
						close(c.events)
						os.Exit(0)
					}
				case 'p':
					c.events <- StateChange{res.Turn, Paused}
					for {
						key := <-c.keyPresses
						if key == 'p' {
							err := client.Call(stubs.KeyPress, req, &res)
							if err != nil {
								fmt.Println("error:", err)
							}
							c.events <- StateChange{res.Turn, Executing}
							break
						}
					}
				}
			}
		}
	}()
	req := stubs.Request{
		World:       World,
		ImageHeight: p.ImageHeight,
		ImageWidth:  p.ImageWidth,
		Turn:        p.Turns,
		Threads:     p.Threads,
	}
	res := new(stubs.Response)
	err = client.Call(stubs.Gameoflife, req, res)
	World = res.World
	c.events <- TurnComplete{res.Turn}
	ticker.Stop()
	filename := fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, res.Turn)
	output(p, c, res.World, filename)
	c.events <- ImageOutputComplete{res.Turn, filename}
	c.events <- FinalTurnComplete{res.Turn, calculateAliveCells(p, World)}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{res.Turn, Quitting}
	client.Close()
	close(c.events)
}

func output(p Params, c distributorChannels, World [][]byte, filename string) {
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- World[y][x]
		}
	}
}

func Exit(turn int, c distributorChannels) {
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var cell []util.Cell
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			if world[i][j] == 255 {
				cell = append(cell, util.Cell{X: j, Y: i})
			}
		}
	}
	return cell
}

func createNewWorld(height, width int) [][]byte {
	World := make([][]byte, height)
	for v := range World {
		World[v] = make([]byte, width)
	}
	return World
}
