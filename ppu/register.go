package ppu

func (p *PPU) LCDC() byte {
	return p.mem.Read(0xFF40)
}

func (p *PPU) LY() byte {
	return p.mem.Read(0xFF44)
}

func (p *PPU) SCX() byte {
	return p.mem.Read(0xFF43)
}

func (p *PPU) SCY() byte {
	return p.mem.Read(0xFF42)
}
func (p *PPU) WX() byte {
	return p.mem.Read(0xFF4B)
}
func (p *PPU) WY() byte {
	return p.mem.Read(0xFF4A)
}

func (p *PPU) STAT() byte {
	return p.mem.Read(0xFF41)
}

func (p *PPU) BGP() byte {
	return p.mem.Read(0xFF47)
}

func (p *PPU) OBP0() byte {
	return p.mem.Read(0xFF48)
}

func (p *PPU) OBP1() byte {
	return p.mem.Read(0xFF49)
}

func (p *PPU) VRAM() []byte {
	return p.mem.RangeInclusive(0x8000, 0x97FF)
}

func (p *PPU) OAM() []byte {
	return p.mem.RangeInclusive(0xFE00, 0xFE9F)
}
