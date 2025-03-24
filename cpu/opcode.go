package cpu

func (c *CPU) ldXNN(reg *byte) {
	nn := c.mem.Read(c.PC)
	*reg = nn
	c.PC++
}

func (c *CPU) addCarry(reg *byte, term byte) {
	old := (*reg)
	carry := (c.F & FLAG_CARRY) >> 4
	sum := uint16(*reg) + uint16(term) + uint16(carry)
	*reg = byte(sum & 0xFF)

	c.F = 0
	if (old&0x0F)+(term&0x0F)+carry > 0x0F {
		c.F |= FLAG_HALFCARRY
	}
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if sum > 0xFF {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) add(reg *byte, term byte) {
	old := (*reg)
	sum := uint16(*reg) + uint16(term)
	*reg = byte(sum & 0xFF)

	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if (old&0x0F)+(term&0x0F) > 0x0F {
		c.F |= FLAG_HALFCARRY
	}
	if sum > 0xFF {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) sub(reg *byte, sub byte) {
	old := (*reg)
	res := int16(*reg) - int16(sub)
	*reg = byte(res & 0xFF)

	c.F = FLAG_SUBTRACT
	if (old & 0x0F) < (sub & 0x0F) {
		c.F |= FLAG_HALFCARRY
	}
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if res < 0 {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) subCarry(reg *byte, sub byte) {
	old := (*reg)
	carry := (c.F & FLAG_CARRY) >> 4
	res := int16(*reg) - int16(sub) - int16(carry)
	*reg = byte(res & 0xFF)

	c.F = FLAG_SUBTRACT
	if (old & 0x0F) < (sub&0x0F)+carry {
		c.F |= FLAG_HALFCARRY
	}
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if res < 0 {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) and(reg *byte, value byte) {
	*reg &= value
	c.F = FLAG_HALFCARRY
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
}

func (c *CPU) xor(reg *byte, value byte) {
	*reg ^= value
	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
}

func (c *CPU) or(reg *byte, value byte) {
	*reg |= value
	c.F = 0
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
}

func (c *CPU) cp(reg byte, value byte) {
	c.F = FLAG_SUBTRACT
	if reg == value {
		c.F |= FLAG_ZERO
	}
	if (reg & 0x0F) < (value & 0x0F) {
		c.F |= FLAG_HALFCARRY
	}
	if reg < value {
		c.F |= FLAG_CARRY
	}
}

func (c *CPU) jr() {
	offset := int8(c.mem.Read(c.PC))
	c.PC++
	c.PC = uint16(int32(c.PC) + int32(offset))
}

func (c *CPU) inc(reg *byte) {
	oldReg := *reg
	(*reg)++
	c.F &= FLAG_CARRY
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if oldReg&0x0F == 0x0F {
		c.F |= FLAG_HALFCARRY
	}
}

func (c *CPU) dec(reg *byte) {
	old := *reg
	(*reg)--

	c.F &= FLAG_CARRY
	c.F |= FLAG_SUBTRACT
	if *reg == 0 {
		c.F |= FLAG_ZERO
	}
	if (old & 0x0F) == 0 {
		c.F |= FLAG_HALFCARRY
	}
}

func (c *CPU) jp() {
	low := c.mem.Read(c.PC)
	high := c.mem.Read(c.PC + 1)

	c.PC = (uint16(high) << 8) | uint16(low)
}
