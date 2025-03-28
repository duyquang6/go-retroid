package gbc

import (
	"log/slog"

	"github.com/duyquang6/go-retroid/cpu"
	"github.com/duyquang6/go-retroid/mmu"
)

type GameBoy struct {
	cpu *cpu.CPU
	mem *mmu.Memory
}

func NewGameBoy() *GameBoy {
	mem := mmu.New()
	cpu := cpu.New(mem)
	return &GameBoy{cpu: cpu, mem: mem}
}

func (gb *GameBoy) LoadROM(rom []uint8) {
	gb.mem.WriteBytes(0, rom)
}

func (gb *GameBoy) Run() {
	slog.Info("Starting emulation...")
	for i := 0; i < 3; i++ { // Run 3 steps for now
		gb.cpu.Step()
	}
}
