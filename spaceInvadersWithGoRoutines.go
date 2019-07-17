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
	register         map[string]piece
	registerChannel  chan msg
	generalChannel   chan msg
	childrenChannels map[string](*chan msg)
}

type msg struct {
	cmd string
	val string
	p   piece
}

var gos uint64

func callGo(f func()) {
	atomic.AddUint64(&gos, 1)
	go f()
}

func (p *piece) conductor() {
	callGo(p.listenThenAct)
	for range time.NewTicker(time.Duration(pieceConductorTickMS) * time.Millisecond).C {
		p.in <- msg{cmd: "Move"}
	}
}

func (p *piece) listenThenAct() {
	for {
		m := <-p.in
		if m.cmd == "Die" {
			p.alive = false
			ms.registerChannel <- msg{cmd: "Remove", p: *p}
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
			ms.generalChannel <- msg{cmd: "Add", val: "Bullet", p: piece{x: p.x, y: p.y, vx: 0, vy: 100}}
		}
		if m.cmd == "Move" {
			var xPixPerBeat = p.vx / 1000 * float64(msConductorTickMS)
			var yPixPerBeat = p.vy / 1000 * float64(msConductorTickMS)
			p.x = p.x + xPixPerBeat
			if p.alive && p.kind == "Alien" {
				if rand.Intn(10000) < 1 {
					ms.generalChannel <- msg{cmd: "Add", val: "Bomb", p: piece{x: p.x, y: p.y, vx: 0, vy: -100}}
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
				ms.registerChannel <- msg{cmd: "Remove", p: *p}
			} else if p.kind == "Bomb" && p.y < 0 {
				p.alive = false
				ms.registerChannel <- msg{cmd: "Remove", p: *p}
			}
			if p.alive {
				ms.registerChannel <- msg{cmd: "Set", p: *p}
			}
		}
	}
}

func (m *motherShip) conductor() {
	populate()
	callGo(ms.listenThenAct)
	var a = 0
	for range time.NewTicker(time.Duration(msConductorTickMS) * time.Millisecond).C {
		a = a + 1
		if a%2 == 0 {
			ms.generalChannel <- msg{cmd: "Display"}
		}
		if a%2 == 0 {
			ms.generalChannel <- msg{cmd: "CheckCollisions"}
		}
		if a%2 == 0 {
			ms.generalChannel <- msg{cmd: "CheckKeys"}
		}
	}
}

func (m *motherShip) listenThenAct() {
	for {
		fmt.Printf("Num Go Routines: %v Num msgs in generalChannel: %v  Num msgs in registerChannel: %v\n", atomic.LoadUint64(&gos), len(ms.generalChannel), len(ms.registerChannel))

		select {
		case message := <-m.registerChannel:
			if message.cmd == "Set" {
				m.register[message.p.name] = piece{name: message.p.name, kind: message.p.kind, x: message.p.x, y: message.p.y, alive: message.p.alive}
			} else if message.cmd == "Remove" {
				delete(m.register, message.p.name)
			}
		case message := <-m.generalChannel:
			if message.cmd == "Add" {
				name := message.val
				if name != "Gun" {
					name = fmt.Sprintf(message.val, time.Now())
				}
				p := piece{name: name, kind: message.val, x: message.p.x, y: message.p.y, alive: true, vx: message.p.vx, vy: message.p.vy, in: make(chan msg)}
				m.childrenChannels[p.name] = &p.in
				callGo(p.conductor)
			}
			if message.cmd == "CheckCollisions" {
				for _, s1 := range m.register {
					if s1.alive && s1.kind == "Bullet" {
						for _, s2 := range m.register {
							if s2.alive && (s2.kind == "Alien" || s2.kind == "Fortress") {
								if (s2.x-s1.x)*(s2.x-s1.x)+(s2.y-s1.y)*(s2.y-s1.y) < 40 {
									*m.childrenChannels[s2.name] <- msg{cmd: "Die"}
									*m.childrenChannels[s1.name] <- msg{cmd: "Die"}
								}
							}
						}
					}
					if s1.alive && s1.kind == "Bomb" {
						for _, s2 := range m.register {
							if s2.alive && (s2.kind == "Gun" || s2.kind == "Fortress") {
								if (s2.x-s1.x)*(s2.x-s1.x)+(s2.y-s1.y)*(s2.y-s1.y) < 40 {
									*m.childrenChannels[s2.name] <- msg{cmd: "Die"}
									*m.childrenChannels[s1.name] <- msg{cmd: "Die"}
								}
							}
						}
					}
				}
			}
			if message.cmd == "CheckKeys" {
				if win != nil {
					if m.childrenChannels["Gun"] != nil {
						if win.Pressed(pixelgl.KeyLeft) {
							*m.childrenChannels["Gun"] <- msg{cmd: "Left"}
						} else if win.Pressed(pixelgl.KeyRight) {
							*m.childrenChannels["Gun"] <- msg{cmd: "Right"}
						} else if win.Pressed(pixelgl.KeySpace) {
							*m.childrenChannels["Gun"] <- msg{cmd: "Shoot"}
						} else {
							*m.childrenChannels["Gun"] <- msg{cmd: "Stop"}
						}
					}
				}
			}
			if message.cmd == "Display" {
				if win != nil {
					var imd = imdraw.New(nil)
					imd.Clear()
					imd.Color = colornames.Limegreen
					for _, s := range m.register {
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
	var spacingPx = 50
	for r := 0; r < 300/spacingPx; r++ {
		for i := 0; i < 400/spacingPx; i++ {
			ms.generalChannel <- msg{cmd: "Add", val: "Alien", p: piece{x: float64(100 + spacingPx*i), y: float64(200 + spacingPx*r), vx: float64(-100 + rand.Intn(1)), vy: 0}}
		}
	}
	for f := 0; f < 3; f++ {
		for c := 0; c < 3; c++ {
			for r := 0; r < 3; r++ {
				ms.generalChannel <- msg{cmd: "Add", val: "Fortress", p: piece{x: float64(100 + f*100 + c*10), y: float64(20 + r*10), vx: 0, vy: 0}}
			}
		}
	}
	ms.generalChannel <- msg{cmd: "Add", val: "Gun", p: piece{x: 100, y: 10, vx: 0, vy: 0}}
}

func start() {
	win, _ = pixelgl.NewWindow(pixelgl.WindowConfig{Title: "Exploring GoRoutines", Bounds: pixel.R(0, 0, float64(512), float64(512)), VSync: true})
	win.SetPos(win.GetPos().Add(pixel.V(0, 1)))
	callGo(ms.conductor)
	<-make(chan bool)
}

var win *pixelgl.Window

var ms = motherShip{register: make(map[string]piece), childrenChannels: make(map[string](*chan msg)), generalChannel: make(chan msg, 20000), registerChannel: make(chan msg, 20000)}

var pieceConductorTickMS int
var msConductorTickMS int

func main() {
	flag.IntVar(&pieceConductorTickMS, "pieceConductor", 10, "Heart Beat for Pieces  MS")
	flag.IntVar(&msConductorTickMS, "msConductorTickMS", 20, "Heart Beat for Mothership MS")

	flag.Parse()
	pixelgl.Run(start)
}
