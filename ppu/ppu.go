package ppu

import "github.com/duyquang6/go-retroid/mmu"

type PPU struct {
	// SharedMem with CPU
	mem *mmu.Memory
}
