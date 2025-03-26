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

func (c *CPU) rl(reg *byte) {
	msb := *reg & 0x80
	*reg = (*reg << 1) | ((c.F & FLAG_CARRY) >> 4)

	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if msb != 0 {
		c.F |= FLAG_CARRY
	}
}
func (c *CPU) rr(reg *byte) {
	lsb := *reg & 0x01
	*reg = (*reg >> 1) | ((c.F & FLAG_CARRY) << 3)

	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if lsb != 0 {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) sla(reg *byte) {
	msb := *reg & 0x80
	*reg <<= 1

	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}

	if msb != 0 {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) sra(reg *byte) {
	lsb := *reg & 0x01
	msb := *reg & 0x80
	*reg = (*reg >> 1) | msb

	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}

	if lsb != 0 {
		c.F |= FLAG_CARRY
	}
}
