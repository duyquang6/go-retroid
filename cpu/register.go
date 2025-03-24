package cpu

const (
	FLAG_ZERO      byte = 0x80
	FLAG_SUBTRACT  byte = 0x40
	FLAG_HALFCARRY byte = 0x20
	FLAG_CARRY     byte = 0x10
)

func (c *CPU) BC() uint16 {
	return uint16(c.B)<<8 | uint16(c.C)
}

func (c *CPU) WriteBC(data uint16) {
	c.B = byte((data & 0xFF00) >> 8)
	c.C = byte(data & 0x00FF)
}

func (c *CPU) HL() uint16 {
	return uint16(c.H)<<8 | uint16(c.L)
}

func (c *CPU) WriteHL(data uint16) {
	c.H = byte((data & 0xFF00) >> 8)
	c.L = byte(data & 0x00FF)
}

func (c *CPU) DE() uint16 {
	return uint16(c.D)<<8 | uint16(c.E)
}

func (c *CPU) WriteDE(data uint16) {
	c.D = byte((data & 0xFF00) >> 8)
	c.E = byte(data & 0x00FF)
}
