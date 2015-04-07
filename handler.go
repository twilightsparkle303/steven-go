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

package main

import (
	"bytes"
	"fmt"
	"math"
	"reflect"

	"github.com/thinkofdeath/steven/protocol"
	"github.com/thinkofdeath/steven/render"
)

type handler map[reflect.Type]reflect.Value

var defaultHandler = handler{}

func init() {
	defaultHandler.Init()
}

func (h handler) Init() {
	v := reflect.ValueOf(h)

	packet := reflect.TypeOf((*protocol.Packet)(nil)).Elem()
	pm := reflect.TypeOf((*pluginMessage)(nil)).Elem()

	for i := 0; i < v.NumMethod(); i++ {
		m := v.Method(i)
		t := m.Type()
		if t.NumIn() != 1 && t.Name() != "Handle" {
			continue
		}
		in := t.In(0)
		if in.AssignableTo(packet) || in.AssignableTo(pm) {
			h[in] = m
		}
	}
}

func (h handler) Handle(packet interface{}) {
	m, ok := h[reflect.TypeOf(packet)]
	if ok {
		m.Call([]reflect.Value{reflect.ValueOf(packet)})
	}
}

func (handler) ServerMessage(msg *protocol.ServerMessage) {
	fmt.Printf("MSG(%d): %s\n", msg.Type, msg.Message.Value)
}

func (handler) Respawn(c *protocol.Respawn) {
	for _, c := range chunkMap {
		c.free()
	}
	chunkMap = map[chunkPosition]*chunk{}
}

func (handler) ChunkData(c *protocol.ChunkData) {
	if c.BitMask == 0 && c.New {
		pos := chunkPosition{int(c.ChunkX), int(c.ChunkZ)}
		c, ok := chunkMap[pos]
		if ok {
			c.free()
			delete(chunkMap, pos)
		}
		return
	}
	go loadChunk(int(c.ChunkX), int(c.ChunkZ), c.Data, c.BitMask, true, c.New)
}

func (handler) ChunkDataBulk(c *protocol.ChunkDataBulk) {
	go func() {
		offset := 0
		data := c.Data
		for _, meta := range c.Meta {
			offset += loadChunk(int(meta.ChunkX), int(meta.ChunkZ), data[offset:], meta.BitMask, c.SkyLight, true)
		}
	}()
}

func (handler) SetBlock(b *protocol.BlockChange) {
	block := GetBlockByCombinedID(uint16(b.BlockID))
	chunkMap.SetBlock(block, b.Location.X(), b.Location.Y(), b.Location.Z())
	chunkMap.UpdateBlock(b.Location.X(), b.Location.Y(), b.Location.Z())
}

func (handler) SetBlockBatch(b *protocol.MultiBlockChange) {
	chunk := chunkMap[chunkPosition{int(b.ChunkX), int(b.ChunkZ)}]
	if chunk == nil {
		return
	}
	for _, r := range b.Records {
		block := GetBlockByCombinedID(uint16(r.BlockID))
		x, y, z := int(r.XZ>>4), int(r.Y), int(r.XZ&0xF)
		chunk.setBlock(block, x, y, z)
		chunkMap.UpdateBlock((chunk.X<<4)+x, y, (chunk.Z<<4)+z)
	}
}

func (handler) JoinGame(j *protocol.JoinGame) {
	ready = true
	sendPluginMessage(&pmMinecraftBrand{
		Brand: "Steven",
	})
}

func (h handler) PluginMessage(p *protocol.PluginMessageClientbound) {
	h.handlePluginMessage(p.Channel, bytes.NewReader(p.Data), false)
}

func (h handler) ServerBrand(b *pmMinecraftBrand) {
	fmt.Printf("The server is running: %s\n", b.Brand)
}

func (handler) Teleport(t *protocol.TeleportPlayer) {
	render.Camera.X = t.X
	render.Camera.Y = t.Y
	render.Camera.Z = t.Z
	render.Camera.Yaw = float64(-t.Yaw) * (math.Pi / 180)
	render.Camera.Pitch = -float64(t.Pitch)*(math.Pi/180) + math.Pi
	writeChan <- &protocol.PlayerPositionLook{
		X:     t.X,
		Y:     t.Y,
		Z:     t.Z,
		Yaw:   t.Yaw,
		Pitch: t.Pitch,
	}
}
