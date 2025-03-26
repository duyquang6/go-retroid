package ppu

// LCD Control bit flags
const (
	LCDC_BG_ENABLE     = 1 << 0
	LCDC_OBJ_ENABLE    = 1 << 1
	LCDC_OBJ_SIZE      = 1 << 2
	LCDC_BG_MAP        = 1 << 3
	LCDC_BG_TILE       = 1 << 4
	LCDC_WINDOW_ENABLE = 1 << 5
	LCDC_WINDOW_MAP    = 1 << 6
	LCDC_LCD_ENABLE    = 1 << 7
)

// LCD Status modes
const (
	MODE_HBLANK = 0
	MODE_VBLANK = 1
	MODE_OAM    = 2
	MODE_VRAM   = 3
)

type PPU struct {
	// LCD Control Register (0xFF40)
	lcdControl byte

	// LCD Status Register (0xFF41)
	lcdStatus byte

	// LCD Position and Scrolling
	scrollY   byte // 0xFF42
	scrollX   byte // 0xFF43
	lyCounter byte // 0xFF44 - Current scanline
	lyCompare byte // 0xFF45
	windowY   byte // 0xFF4A
	windowX   byte // 0xFF4B

	// LCD Color Palettes
	bgPalette   byte // 0xFF47
	objPalette0 byte // 0xFF48
	objPalette1 byte // 0xFF49

	// Video RAM
	vram [8192]byte // 8KB Video RAM
	oam  [160]byte  // Object Attribute Memory

	// Internal timing
	clock int
	mode  byte // Current PPU mode

	// Frame buffer
	frameBuffer [160 * 144]byte
}

// NewPPU creates a new PPU instance
func NewPPU() *PPU {
	return &PPU{
		lcdControl: 0x91, // Default value
		lyCounter:  0,
		mode:       MODE_OAM,
	}
}

// Step advances the PPU state
func (p *PPU) Step(cycles int) {
	if p.lcdControl&LCDC_LCD_ENABLE == 0 {
		return
	}

	p.clock += cycles

	switch p.mode {
	case MODE_OAM: // Searching OAM - 80 cycles
		if p.clock >= 80 {
			p.mode = MODE_VRAM
			p.clock -= 80
		}

	case MODE_VRAM: // Reading VRAM - 172 cycles
		if p.clock >= 172 {
			p.renderScanline()
			p.mode = MODE_HBLANK
			p.clock -= 172
		}

	case MODE_HBLANK: // HBlank - 204 cycles
		if p.clock >= 204 {
			p.clock -= 204
			p.lyCounter++

			if p.lyCounter == 144 {
				p.mode = MODE_VBLANK
			} else {
				p.mode = MODE_OAM
			}
		}

	case MODE_VBLANK: // VBlank - 4560 cycles (10 scanlines)
		if p.clock >= 456 {
			p.clock -= 456
			p.lyCounter++

			if p.lyCounter > 153 {
				p.lyCounter = 0
				p.mode = MODE_OAM
			}
		}
	}
}

// renderScanline renders a single scanline
func (p *PPU) renderScanline() {
	if p.lcdControl&LCDC_BG_ENABLE != 0 {
		p.renderBackground()
	}
	if p.lcdControl&LCDC_OBJ_ENABLE != 0 {
		p.renderSprites()
	}
}

// Read reads a byte from PPU memory
func (p *PPU) Read(addr uint16) byte {
	switch {
	case addr >= 0x8000 && addr <= 0x9FFF:
		return p.vram[addr-0x8000]
	case addr >= 0xFE00 && addr <= 0xFE9F:
		return p.oam[addr-0xFE00]
	case addr == 0xFF40:
		return p.lcdControl
	case addr == 0xFF41:
		return p.lcdStatus
	case addr == 0xFF42:
		return p.scrollY
	case addr == 0xFF43:
		return p.scrollX
	case addr == 0xFF44:
		return p.lyCounter
	}
	return 0
}

// Write writes a byte to PPU memory
func (p *PPU) Write(addr uint16, val byte) {
	switch {
	case addr >= 0x8000 && addr <= 0x9FFF:
		p.vram[addr-0x8000] = val
	case addr >= 0xFE00 && addr <= 0xFE9F:
		p.oam[addr-0xFE00] = val
	case addr == 0xFF40:
		p.lcdControl = val
	case addr == 0xFF41:
		p.lcdStatus = val
	case addr == 0xFF42:
		p.scrollY = val
	case addr == 0xFF43:
		p.scrollX = val
	}
}
