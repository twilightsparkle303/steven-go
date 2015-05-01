// Copyright 2015 Matthew Collins
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package steven

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image/png"
	"math"
	"strings"

	"github.com/go-gl/glfw/v3.1/glfw"
	"github.com/thinkofdeath/steven/chat"
	"github.com/thinkofdeath/steven/protocol"
	"github.com/thinkofdeath/steven/render"
	"github.com/thinkofdeath/steven/resource"
	"github.com/thinkofdeath/steven/ui"
	"github.com/thinkofdeath/steven/ui/scene"
)

type serverList struct {
	scene *scene.Type
	logo  uiLogo

	servers []*serverListItem
}

type serverListItem struct {
	*scene.Type

	container *ui.Container
	offset    float64
	id        string
}

func newServerList() *serverList {
	sl := &serverList{
		scene: scene.New(true),
	}
	Client.scene.Hide()
	sl.logo.init(sl.scene)
	window.SetScrollCallback(sl.onScroll)

	sl.scene.AddDrawable(
		ui.NewText("Steven - "+resource.ResourcesVersion, 5, 5, 255, 255, 255).Attach(ui.Bottom, ui.Left),
	)
	sl.scene.AddDrawable(
		ui.NewText("Not affiliated with Mojang/Minecraft", 5, 5, 255, 200, 200).Attach(ui.Bottom, ui.Right),
	)

	sl.redraw()

	refresh := ui.NewButton(300, -12-50, 100, 22)
	refresh, txt := newButtonText("Refresh", 300, -50-15, 100, 30)
	sl.scene.AddDrawable(refresh.Attach(ui.Center, ui.Middle))
	sl.scene.AddDrawable(txt)
	refresh.ClickFunc = sl.redraw

	return sl
}

func (sl *serverList) onScroll(w *glfw.Window, xoff float64, yoff float64) {
	if len(sl.servers) == 0 {
		return
	}
	diff := yoff / 10
	if s := sl.servers[len(sl.servers)-1]; s.offset+diff <= 2 {
		diff = 2 - s.offset
	}
	if s := sl.servers[0]; s.offset+diff >= 0 {
		diff = -s.offset
	}
	for _, s := range sl.servers {
		s.offset += diff
		s.updatePosition()
	}
}

func (si *serverListItem) updatePosition() {
	if si.offset < 0 {
		si.container.Y = si.offset * 200
		si.container.X = -math.Abs(si.offset) * 300
	} else if si.offset >= 2 {
		si.container.Y = si.offset * 100
		si.container.X = -math.Abs(si.offset-2) * 300
	} else {
		si.container.X = 0
		si.container.Y = si.offset * 100
	}
}

func (sl *serverList) redraw() {
	for _, s := range sl.servers {
		s.Hide()
		render.FreeIcon(s.id)
	}
	sl.servers = sl.servers[:0]
	for i, s := range Config.Servers {
		sc := scene.New(true)
		container := (&ui.Container{
			X: 0, Y: float64(i) * 100, W: 700, H: 100,
		}).Attach(ui.Center, ui.Middle)
		r := make([]byte, 20)
		rand.Read(r)
		si := &serverListItem{
			Type:      sc,
			container: container,
			offset:    float64(i),
			id:        "servericon:" + string(r),
		}
		si.updatePosition()
		sl.servers = append(sl.servers, si)

		bck := ui.NewImage(render.GetTexture("solid"), 0, 0, 700, 100, 0, 0, 1, 1, 0, 0, 0).Attach(ui.Top, ui.Left)
		bck.A = 100
		bck.Parent = container
		sc.AddDrawable(bck)
		txt := ui.NewText(s.Name, 90+10, 5, 255, 255, 255).Attach(ui.Top, ui.Left)
		txt.Parent = container
		sc.AddDrawable(txt)

		icon := ui.NewImage(render.GetTexture("misc/unknown_server"), 5, 5, 90, 90, 0, 0, 1, 1, 255, 255, 255).
			Attach(ui.Top, ui.Left)
		icon.Parent = container
		sc.AddDrawable(icon)

		ping := ui.NewImage(render.GetTexture("gui/icons"), 5, 5, 20, 16, 0, 56/256.0, 10/256.0, 8/256.0, 255, 255, 255).
			Attach(ui.Top, ui.Right)
		ping.Parent = container
		sc.AddDrawable(ping)

		players := ui.NewText("???", 30, 5, 255, 255, 255).
			Attach(ui.Top, ui.Right)
		players.Parent = container
		sc.AddDrawable(players)

		msg := &chat.TextComponent{Text: "Connecting..."}
		motd := ui.NewFormattedWidth(chat.AnyComponent{msg}, 90+10, 5+18, 700-(90+10+5)).Attach(ui.Top, ui.Left)
		motd.Parent = container
		sc.AddDrawable(motd)
		s := s
		go func() {
			sl.pingServer(s.Address, motd, icon, si.id, ping, players)
		}()
		container.ClickFunc = func() {
			sl.connect(s.Address)
		}
		container.HoverFunc = func(over bool) {
			if over {
				bck.A = 200
			} else {
				bck.A = 100
			}
		}
		sc.AddDrawable(container)
	}
}

func (sl *serverList) pingServer(addr string, motd *ui.Formatted,
	icon *ui.Image, id string, ping *ui.Image, players *ui.Text) {
	conn, err := protocol.Dial(addr)
	if err != nil {
		syncChan <- func() {
			msg := &chat.TextComponent{Text: err.Error()}
			msg.Color = chat.Red
			motd.Update(chat.AnyComponent{msg})
		}
		return
	}
	resp, pingTime, err := conn.RequestStatus()
	syncChan <- func() {
		if err != nil {
			msg := &chat.TextComponent{Text: err.Error()}
			msg.Color = chat.Red
			motd.Update(chat.AnyComponent{msg})
			return
		}
		y := 0.0
		pt := pingTime.Seconds() / 1000
		switch {
		case pt <= 75:
			y = 16 / 256.0
		case pt <= 150:
			y = 24 / 256.0
		case pt <= 225:
			y = 32 / 256.0
		case pt <= 350:
			y = 40 / 256.0
		case pt < 999:
			y = 48 / 256.0
		default:
			y = 56 / 256.0
		}
		ping.TY = y

		players.Update(fmt.Sprintf("%d/%d", resp.Players.Online, resp.Players.Max))

		desc := resp.Description
		chat.ConvertLegacy(desc)
		motd.Update(desc)

		if strings.HasPrefix(resp.Favicon, "data:image/png;base64,") {
			favicon := resp.Favicon[len("data:image/png;base64,"):]
			data, err := base64.StdEncoding.DecodeString(favicon)
			if err != nil {
				fmt.Printf("error base64 decoding favicon: %s\n", err)
				return
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				fmt.Printf("error decoding favicon: %s\n", err)
				return
			}
			render.AddIcon(id, img)
			icon.Texture = render.Icon(id)
		}

	}
}

func (sl *serverList) connect(s string) {
	server = s
	initClient()
	Client.init()
	connect()
	setScreen(nil)
}

func (sl *serverList) hover(x, y float64, w, h int) {
	ui.Hover(x, y, w, h)
}
func (sl *serverList) click(x, y float64, w, h int) {
	ui.Click(x, y, w, h)
}
func (sl *serverList) tick(delta float64) {
	sl.logo.tick(delta)
}

func (sl *serverList) remove() {
	window.SetScrollCallback(onScroll)
	sl.scene.Hide()
	for _, s := range sl.servers {
		s.Hide()
		render.FreeIcon(s.id)
	}
}
