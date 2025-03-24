package cpu

func (c *CPU) ldXNN(reg *byte) {
	nn := c.mem.Read(c.PC)
	*reg = nn
	c.PC++
}

func (c *CPU) addCarry(reg *byte, term byte) {
	old := (*reg)
	carry := (c.F & CARRY) >> 4
	sum := uint16(*reg) + uint16(term) + uint16(carry)
	*reg = byte(sum & 0xFF)

	c.F = 0
	if (old&0x0F)+(term&0x0F)+carry > 0x0F {
		c.F |= HALFCARRY
	}
	if *reg == 0 {
		c.F |= ZERO
	}
	if sum > 0xFF {
		c.F |= CARRY
	}
}

func (c *CPU) add(reg *byte, term byte) {
	old := (*reg)
	sum := uint16(*reg) + uint16(term)
	*reg = byte(sum & 0xFF)

	c.F = 0
	if (old&0x0F)+(term&0x0F) > 0x0F {
		c.F |= HALFCARRY
	}
	if *reg == 0 {
		c.F |= ZERO
	}
	if sum > 0xFF {
		c.F |= CARRY
	}
}

func (c *CPU) sub(reg *byte, sub byte) {
	old := (*reg)
	res := int16(*reg) - int16(sub)
	*reg = byte(res & 0xFF)

	c.F = SUB
	if (old & 0x0F) < (sub & 0x0F) {
		c.F |= HALFCARRY
	}
	if *reg == 0 {
		c.F |= ZERO
	}
	if res < 0 {
		c.F |= CARRY
	}
}

func (c *CPU) subCarry(reg *byte, sub byte) {
	old := (*reg)
	carry := (c.F & CARRY) >> 4
	res := int16(*reg) - int16(sub) - int16(carry)
	*reg = byte(res & 0xFF)

	c.F = SUB
	if (old & 0x0F) < (sub&0x0F)+carry {
		c.F |= HALFCARRY
	}
	if *reg == 0 {
		c.F |= ZERO
	}
	if res < 0 {
		c.F |= CARRY
	}
}

func (c *CPU) and(reg *byte, value byte) {
	*reg &= value
	c.F = HALFCARRY
	if *reg == 0 {
		c.F |= ZERO
	}
}

func (c *CPU) xor(reg *byte, value byte) {
	*reg ^= value
	c.F = 0
	if *reg == 0 {
		c.F |= ZERO
	}
}

func (c *CPU) or(reg *byte, value byte) {
	*reg |= value
	c.F = 0
	if *reg == 0 {
		c.F |= ZERO
	}
}

func (c *CPU) cp(reg byte, value byte) {
	c.F = SUB
	if reg == value {
		c.F |= ZERO
	}
	if (reg & 0x0F) < (value & 0x0F) {
		c.F |= HALFCARRY
	}
	if reg < value {
		c.F |= CARRY
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
	c.F &= CARRY
	if *reg == 0 {
		c.F |= ZERO
	}
	if oldReg&0x0F == 0x0F {
		c.F |= HALFCARRY
	}
}

func (c *CPU) dec(reg *byte) {
	old := *reg
	(*reg)--

	c.F &= CARRY
	c.F |= SUB
	if *reg == 0 {
		c.F |= ZERO
	}
	if (old & 0x0F) == 0 {
		c.F |= HALFCARRY
	}
}
