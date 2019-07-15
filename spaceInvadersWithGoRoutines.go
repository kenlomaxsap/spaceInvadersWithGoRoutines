package main

import (
	"flag"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

type piece struct {
	kind  string
	name  string
	x     float64
	y     float64
	alive bool
	vx    float64
	vy    float64
	in    chan msg
}

type motherShip struct {
	outs      map[string](*chan msg)
	input1    chan msg
	input2    chan msg
	_register map[string]piece
}

type msg struct {
	cmd  string
	kind string
	p    piece
}

var gos uint64

func callGo(f func()) {
	atomic.AddUint64(&gos, 1)
	go f()
}

func (p *piece) heartBeat() {
	callGo(p.listen)
	for range time.NewTicker(time.Duration(flagPieceHeartBeatMS) * time.Millisecond).C {
		p.in <- msg{cmd: "Move"}
	}
}

func (p *piece) listen() {
	for {
		m := <-p.in
		if m.cmd == "Die" {
			p.alive = false
			ms.input2 <- msg{cmd: "Remove", p: *p}
			if p.kind == "Gun" {
				fmt.Println("Damn and blast")
				<-make(chan bool)
			}
		} else if m.cmd == "Left" {
			p.vx = -50
		} else if m.cmd == "Stop" {
			p.vx = 0
		} else if m.cmd == "Right" {
			p.vx = 50
		} else if m.cmd == "Shoot" {
			ms.input1 <- msg{cmd: "Add", kind: "Bullet", p: piece{x: p.x, y: p.y, vx: 0, vy: 100}}
		}
		if m.cmd == "Move" {
			var xPixPerBeat = p.vx / 1000 * float64(flagMotherShipHeartBeatMS)
			var yPixPerBeat = p.vy / 1000 * float64(flagMotherShipHeartBeatMS)
			p.x = p.x + xPixPerBeat
			if p.alive && p.kind == "Alien" {
				if rand.Intn(10000) < 1 {
					ms.input1 <- msg{cmd: "Add", kind: "Bomb", p: piece{x: p.x, y: p.y, vx: 0, vy: -100}}
				}
				if p.x > 500 || p.x < 10 {
					p.vx = -p.vx
					p.y = p.y - 10
				}
			} else {
				p.y = p.y + yPixPerBeat
			}
			if p.kind == "Bullet" && p.y > 600 {
				p.alive = false
				ms.input2 <- msg{cmd: "Remove", p: *p}
			} else if p.kind == "Bomb" && p.y < 0 {
				p.alive = false
				ms.input2 <- msg{cmd: "Remove", p: *p}
			}
			if p.alive {
				ms.input2 <- msg{cmd: "Set", p: *p}
			}
		}
	}
}

func (m *motherShip) heartBeat() {
	populate()
	callGo(ms.listen)
	var a = 0
	for range time.NewTicker(time.Duration(flagMotherShipHeartBeatMS) * time.Millisecond).C {
		a = a + 1
		if a%2 == 0 {
			ms.input1 <- msg{cmd: "Display"}
		}
		if a%2 == 0 {
			ms.input1 <- msg{cmd: "CheckCollisions"}
		}
		if a%2 == 0 {
			ms.input1 <- msg{cmd: "CheckKeys"}
		}
	}
}

func (m *motherShip) listen() {
	for {
		fmt.Printf("Num Go Routines: %v Num msgs in Input1: %v  Num msgs in Input2: %v\n", atomic.LoadUint64(&gos), len(ms.input1), len(ms.input2))

		select {
		case message := <-m.input2:
			if message.cmd == "Set" {
				m._register[message.p.name] = piece{name: message.p.name, kind: message.p.kind, x: message.p.x, y: message.p.y, alive: message.p.alive}
			} else if message.cmd == "Remove" {
				delete(m._register, message.p.name)
			}
		case message := <-m.input1:
			if message.cmd == "Add" {
				name := message.kind
				if name != "Gun" {
					name = fmt.Sprintf(message.kind, time.Now())
				}
				p := piece{name: name, kind: message.kind, x: message.p.x, y: message.p.y, alive: true, vx: message.p.vx, vy: message.p.vy, in: make(chan msg)}
				m.outs[p.name] = &p.in
				callGo(p.heartBeat)
			}
			if message.cmd == "CheckCollisions" {
				for _, s1 := range m._register {
					if s1.alive && s1.kind == "Bullet" {
						for _, s2 := range m._register {
							if s2.alive && (s2.kind == "Alien" || s2.kind == "Fortress") {
								if (s2.x-s1.x)*(s2.x-s1.x)+(s2.y-s1.y)*(s2.y-s1.y) < 40 {
									*m.outs[s2.name] <- msg{cmd: "Die"}
									*m.outs[s1.name] <- msg{cmd: "Die"}
								}
							}
						}
					}
					if s1.alive && s1.kind == "Bomb" {
						for _, s2 := range m._register {
							if s2.alive && (s2.kind == "Gun" || s2.kind == "Fortress") {
								if (s2.x-s1.x)*(s2.x-s1.x)+(s2.y-s1.y)*(s2.y-s1.y) < 40 {
									*m.outs[s2.name] <- msg{cmd: "Die"}
									*m.outs[s1.name] <- msg{cmd: "Die"}
								}
							}
						}
					}
				}
			}
			if message.cmd == "CheckKeys" {
				if win != nil {
					if m.outs["Gun"] != nil {
						if win.Pressed(pixelgl.KeyLeft) {
							*m.outs["Gun"] <- msg{cmd: "Left"}
						} else if win.Pressed(pixelgl.KeyRight) {
							*m.outs["Gun"] <- msg{cmd: "Right"}
						} else if win.Pressed(pixelgl.KeySpace) {
							*m.outs["Gun"] <- msg{cmd: "Shoot"}
						} else {
							*m.outs["Gun"] <- msg{cmd: "Stop"}
						}
					}
				}
			}
			if message.cmd == "Display" {
				if win != nil {
					var imd = imdraw.New(nil)
					imd.Clear()
					imd.Color = colornames.Limegreen
					for _, s := range m._register {
						if s.alive {
							imd.Push(pixel.V(float64(s.x), float64(s.y)))
							imd.Circle(3, 1)
						}
					}
					win.Clear(colornames.Black)
					imd.Draw(win)
					win.Update()
				}
			}
		}
	}
}

func populate() {
	for r := 0; r < 300/flagSpacingPx; r++ {
		for i := 0; i < 400/flagSpacingPx; i++ {
			ms.input1 <- msg{cmd: "Add", kind: "Alien", p: piece{x: float64(100 + flagSpacingPx*i), y: float64(200 + flagSpacingPx*r), vx: float64(-100 + rand.Intn(1)), vy: 0}}
		}
	}
	for f := 0; f < 3; f++ {
		for c := 0; c < 3; c++ {
			for r := 0; r < 3; r++ {
				ms.input1 <- msg{cmd: "Add", kind: "Fortress", p: piece{x: float64(100 + f*100 + c*10), y: float64(20 + r*10), vx: 0, vy: 0}}
			}
		}
	}
	ms.input1 <- msg{cmd: "Add", kind: "Gun", p: piece{x: 100, y: 10, vx: 0, vy: 0}}
}

func start() {
	win, _ = pixelgl.NewWindow(pixelgl.WindowConfig{Title: "Exploring GoRoutines", Bounds: pixel.R(0, 0, float64(flagWidthPx), float64(flagHeightPx)), VSync: true})
	win.SetPos(win.GetPos().Add(pixel.V(0, 1)))
	callGo(ms.heartBeat)
	<-make(chan bool)
}

var win *pixelgl.Window

var ms = motherShip{_register: make(map[string]piece), outs: make(map[string](*chan msg)), input1: make(chan msg, 20000), input2: make(chan msg, 20000)}

var flagPieceHeartBeatMS int
var flagMotherShipHeartBeatMS int
var flagWidthPx int
var flagHeightPx int
var flagSpacingPx int

func main() {
	flag.IntVar(&flagPieceHeartBeatMS, "flagPieceHeartBeatMS", 10, "Heart Beat for Pieces  MS")
	flag.IntVar(&flagMotherShipHeartBeatMS, "flagMotherShipHeartBeatMS", 20, "Heart Beat for Mothership MS")
	flag.IntVar(&flagWidthPx, "flagWidthPx", 500, "Width of screen Px")
	flag.IntVar(&flagHeightPx, "flagHeightPx", 400, "Height of screen Px")
	flag.IntVar(&flagSpacingPx, "flagSpacingPx", 40, "Width between aliens Px")

	flag.Parse()
	pixelgl.Run(start)
	// callGorun conc0712.callGo-flagPieceHeartBeatMS 10 -flagMotherShipHeartBeatMS 10 -flagWidthPx 600 -flagHeightPx 500 -flagSpacingPx 40
}
