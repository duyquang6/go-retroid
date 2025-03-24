package cpu

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/duyquang6/gboy/mmu"
)

const (
	ZERO      byte = 0x80
	SUB       byte = 0x40
	HALFCARRY byte = 0x20
	CARRY     byte = 0x10
)

type CPU struct {
	// Accumulator Register (Fast processing ALU)
	A byte
	// Flag register: 0x80 = Zero, 0x40 = Subtract, 0x20 = Half Carry, 0x10 = carry
	F byte

	// General purpose
	B, C, D, E, H, L byte

	PC, SP uint16

	mem *mmu.Memory

	stopped bool
}

func New(mem *mmu.Memory) *CPU {
	return &CPU{
		mem: mem,
	}
}

func (c *CPU) Fetch() byte {
	opcode := c.mem.Read(c.PC)
	c.PC++

	return opcode
}

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

func (c *CPU) ldXNN(reg *byte) {
	nn := c.mem.Read(c.PC)
	*reg = nn
	c.PC++
}

func (c *CPU) Execute(opcode byte) {
	switch opcode {
	// 8 bit instruction
	case 0x00: // NOP, do nothing
	case 0x01: // LD BC, d16
		c.B = c.mem.Read(c.PC + 1)
		c.C = c.mem.Read(c.PC)
		c.PC += 2
	case 0x02: // LD (BC), A
		c.mem.Write(c.BC(), c.A)
	case 0x03: // INC BC
		c.WriteBC(c.BC() + 1)
	case 0x04: // INC B
		c.inc(&c.B)
	case 0x05: // DEC B
		c.dec(&c.B)
	case 0x06: // LD B,nn
		c.ldXNN(&c.B)
	case 0x07: // RLCA
		msb := c.A & 0xF0
		c.A <<= 1

		c.F = 0
		if msb == 0x10 {
			c.F |= CARRY
			c.A |= 0x1
		}
	case 0x08: // LD (a16), SP
		c.mem.Write(c.PC, byte(c.SP&0x00FF))
		c.mem.Write(c.PC+1, byte((c.SP&0xFF00)>>8))
		c.PC += 2
	case 0x09: // ADD HL, BC
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.BC())
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(c.BC()&0x00FF) > 0x00FF {
			c.F |= HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= CARRY
		}
	case 0x0A: // LD A, (BC)
		c.A = c.mem.Read(c.BC())
	case 0x0B: // DEC BC
		c.WriteBC(c.BC() - 1)
	case 0x0C: // INC C
		c.inc(&c.C)
	case 0x0D: // DEC C
		c.dec(&c.C)
	case 0x0E: // LD C, d8
		c.ldXNN(&c.C)
	case 0x0F: // RRCA
		lsb := c.A & 0x01
		c.A >>= 1

		c.F = 0
		if lsb == 0x01 {
			c.F |= CARRY
			c.A |= 0x1 << 7
		}

	// 0x1X
	case 0x10: // STOP
		c.stopped = true
		c.PC++
		slog.Info("CPU stopped, awaiting interrupt")
	case 0x11: // LD DE, d16
		c.D = c.mem.Read(c.PC + 1)
		c.E = c.mem.Read(c.PC)
		c.PC += 2
	case 0x12: // LD (DE), A
		c.mem.Write(c.DE(), c.A)
	case 0x13: // INC DE
		c.WriteDE(c.DE() + 1)
	case 0x14: // INC D
		c.inc(&c.D)
	case 0x15: // DEC D
		c.dec(&c.D)
	case 0x16: // LD D, d8
		c.ldXNN(&c.D)
	case 0x17: // RLA
		oldA := c.A
		c.A <<= 1
		if c.F&CARRY > 0 {
			c.A |= 0x01
		}

		c.F = 0
		if oldA&0x80 > 0 {
			c.F = CARRY
		}
	case 0x18: // JR s8
		c.jr()
	case 0x19: // ADD HL, DE
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.DE())
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(c.DE()&0x00FF) > 0x00FF {
			c.F |= HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= CARRY
		}
	case 0x1A: // LD A, (DE)
		c.A = c.mem.Read(c.DE())
	case 0x1B: // DEC DE
		c.WriteBC(c.DE() - 1)
	case 0x1C: // INC E
		c.inc(&c.E)
	case 0x1D: // DEC E
		c.dec(&c.E)
	case 0x1E: // LD E,d8
		c.ldXNN(&c.E)
	case 0x1F: // RRA
		oldA := c.A
		c.A >>= 1
		if c.F&CARRY > 0 {
			c.A |= 0x80
		}

		c.F = 0
		if oldA&0x01 > 0 {
			c.F = CARRY
		}

	// 0x2X
	case 0x20: // JR NZ, s8
		if c.F&ZERO == 0 {
			c.jr()
		}
	case 0x21: // LD HL,d16
		c.H = c.mem.Read(c.PC + 1)
		c.L = c.mem.Read(c.PC)
		c.PC += 2
	case 0x22: // LD (HL+),A
		c.mem.Write(c.HL(), c.A)
		c.WriteHL(c.HL() + 1)
	case 0x23: // INC HL
		c.WriteHL(c.HL() + 1)
	case 0x24: // INC H
		c.inc(&c.H)
	case 0x25: // DEC H
		c.dec(&c.H)
	case 0x26: // LD H,d8
		c.ldXNN(&c.H)
	case 0x27: // DAA
		if c.F&SUB == 0 {
			// Addition
			if (c.A&0x0F) > 9 || (c.F&HALFCARRY) != 0 {
				c.A += 0x06
			}
			if c.A > 0x99 || (c.F&CARRY) != 0 {
				c.A += 0x60
				c.F |= CARRY
			}
		} else {
			if (c.F & HALFCARRY) != 0 {
				c.A -= 0x06
			}
			if (c.F & CARRY) != 0 {
				c.A -= 0x60
			}
		}
		// Reset H to 0
		c.F &= ^HALFCARRY

		if c.A == 0 {
			c.F |= ZERO
		} else {
			c.F &= ^ZERO
		}
	case 0x28: // JR Z,s8
		if c.F&ZERO != 0 {
			c.jr()
		}
	case 0x29: // ADD HL,HL
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.HL())
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(old&0x00FF) > 0x00FF {
			c.F |= HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= CARRY
		}
	case 0x2A: // LD A,(HL+)
		c.A = c.mem.Read(c.HL())
		c.WriteHL(c.HL() + 1)
	case 0x2B: // DEC HL
		c.WriteHL(c.HL() - 1)
	case 0x2C: // INC L
		c.inc(&c.L)
	case 0x2D: // DEC L
		c.dec(&c.L)
	case 0x2E: // LD L,d8
		c.ldXNN(&c.L)
	case 0x2F: // CPL: complement 1 of A
		c.A = ^c.A
		c.F |= HALFCARRY | SUB

	// 0x3X
	case 0x30: // JR NC, s8
		if (c.F & CARRY) != 0 {
			c.jr()
		}
	case 0x31: // LD SP,d16
		low := c.mem.Read(c.PC)
		high := c.mem.Read(c.PC + 1)
		c.SP = uint16(high)<<8 | uint16(low)
		c.PC += 2
	case 0x32: // LD (HL-),A
		c.mem.Write(c.HL(), c.A)
		c.WriteHL(c.HL() - 1)
	case 0x33: // INC SP
		c.SP++
	case 0x34: // INC (HL)
		val := c.mem.Read(c.HL())
		old := val
		val++
		c.mem.Write(c.HL(), val)

		c.F &= 0x1F
		if val == 0 {
			c.F |= ZERO
		}
		if old&0x0F == 0x0F {
			c.F |= HALFCARRY
		}
	case 0x35: // DEC (HL)
		val := c.mem.Read(c.HL())
		old := val
		val--
		c.mem.Write(c.HL(), val)

		if val == 0 {
			c.F |= ZERO
		}
		c.F |= SUB
		if old&0x0F == 0 {
			c.F |= HALFCARRY
		}
	case 0x36: // LD (HL),d8
		val := c.mem.Read(c.PC)
		c.mem.Write(c.HL(), val)
		c.PC++
	case 0x37: // SCF
		c.F = (c.F & ZERO) | CARRY
	case 0x38: // JR C,s8
		if c.F&CARRY != 0 {
			c.jr()
		}
	case 0x39: // ADD HL,SP
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.SP)
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(uint16(c.SP)&0x00FF) > 0x00FF {
			c.F |= HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= CARRY
		}
	case 0x3A: // LD A,(HL-)
		c.A = c.mem.Read(c.HL())
		c.WriteHL(c.HL() - 1)
	case 0x3B: // DEC SP
		c.SP--
	case 0x3C: // INC A
		c.inc(&c.A)
	case 0x3D: // DEC A
		c.dec(&c.A)
	case 0x3E: // LD A,d8
		c.ldXNN(&c.A)
	case 0x3F: // CCF (Complement Carry Flag)
		c.F = (c.F ^ CARRY) & (ZERO | CARRY)

	// 0x4X - Load instructions B
	case 0x40: // LD B,B
		// NOP effectively
	case 0x41: // LD B,C
		c.B = c.C
	case 0x42: // LD B,D
		c.B = c.D
	case 0x43: // LD B,E
		c.B = c.E
	case 0x44: // LD B,H
		c.B = c.H
	case 0x45: // LD B,L
		c.B = c.L
	case 0x46: // LD B,(HL)
		c.B = c.mem.Read(c.HL())
	case 0x47: // LD B,A
		c.B = c.A

	// 0x5X - Load instructions C
	case 0x48: // LD C,B
		c.C = c.B
	case 0x49: // LD C,C
		// NOP effectively
	case 0x4A: // LD C,D
		c.C = c.D
	case 0x4B: // LD C,E
		c.C = c.E
	case 0x4C: // LD C,H
		c.C = c.H
	case 0x4D: // LD C,L
		c.C = c.L
	case 0x4E: // LD C,(HL)
		c.C = c.mem.Read(c.HL())
	case 0x4F: // LD C,A
		c.C = c.A

	// 0x5X - Load instructions D
	case 0x50: // LD D,B
		c.D = c.B
	case 0x51: // LD D,C
		c.D = c.C
	case 0x52: // LD D,D
		// NOP effectively
	case 0x53: // LD D,E
		c.D = c.E
	case 0x54: // LD D,H
		c.D = c.H
	case 0x55: // LD D,L
		c.D = c.L
	case 0x56: // LD D,(HL)
		c.D = c.mem.Read(c.HL())
	case 0x57: // LD D,A
		c.D = c.A

	// 0x5X - Load instructions E
	case 0x58: // LD E,B
		c.E = c.B
	case 0x59: // LD E,C
		c.E = c.C
	case 0x5A: // LD E,D
		c.E = c.D
	case 0x5B: // LD E,E
		// NOP effectively
	case 0x5C: // LD E,H
		c.E = c.H
	case 0x5D: // LD E,L
		c.E = c.L
	case 0x5E: // LD E,(HL)
		c.E = c.mem.Read(c.HL())
	case 0x5F: // LD E,A
		c.E = c.A

	// 0x6X - Load instructions H
	case 0x60: // LD H,B
		c.H = c.B
	case 0x61: // LD H,C
		c.H = c.C
	case 0x62: // LD H,D
		c.H = c.D
	case 0x63: // LD H,E
		c.H = c.E
	case 0x64: // LD H,H
		// NOP effectively
	case 0x65: // LD H,L
		c.H = c.L
	case 0x66: // LD H,(HL)
		c.H = c.mem.Read(c.HL())
	case 0x67: // LD H,A
		c.H = c.A

	// 0x6X - Load instructions L
	case 0x68: // LD L,B
		c.L = c.B
	case 0x69: // LD L,C
		c.L = c.C
	case 0x6A: // LD L,D
		c.L = c.D
	case 0x6B: // LD L,E
		c.L = c.E
	case 0x6C: // LD L,H
		c.L = c.H
	case 0x6D: // LD L,L
		// NOP effectively
	case 0x6E: // LD L,(HL)
		c.L = c.mem.Read(c.HL())
	case 0x6F: // LD L,A
		c.L = c.A

	// 0x7X - Load instructions to/from memory and A
	case 0x70: // LD (HL),B
		c.mem.Write(c.HL(), c.B)
	case 0x71: // LD (HL),C
		c.mem.Write(c.HL(), c.C)
	case 0x72: // LD (HL),D
		c.mem.Write(c.HL(), c.D)
	case 0x73: // LD (HL),E
		c.mem.Write(c.HL(), c.E)
	case 0x74: // LD (HL),H
		c.mem.Write(c.HL(), c.H)
	case 0x75: // LD (HL),L
		c.mem.Write(c.HL(), c.L)
	case 0x76: // HALT
		c.stopped = true
	case 0x77: // LD (HL),A
		c.mem.Write(c.HL(), c.A)
	case 0x78: // LD A,B
		c.A = c.B
	case 0x79: // LD A,C
		c.A = c.C
	case 0x7A: // LD A,D
		c.A = c.D
	case 0x7B: // LD A,E
		c.A = c.E
	case 0x7C: // LD A,H
		c.A = c.H
	case 0x7D: // LD A,L
		c.A = c.L
	case 0x7E: // LD A,(HL)
		c.A = c.mem.Read(c.HL())
	case 0x7F: // LD A,A
		// NOP effectively

	// 0x8X - ADD instructions
	case 0x80: // ADD A,B
		c.addReg8(&c.A, c.B)
	case 0x81: // ADD A,C
		c.addReg8(&c.A, c.C)
	case 0x82: // ADD A,D
		c.addReg8(&c.A, c.D)
	case 0x83: // ADD A,E
		c.addReg8(&c.A, c.E)
	case 0x84: // ADD A,H
		c.addReg8(&c.A, c.H)
	case 0x85: // ADD A,L
		c.addReg8(&c.A, c.L)
	case 0x86: // ADD A,(HL)
		c.addReg8(&c.A, c.mem.Read(c.HL()))
	case 0x87: // ADD A,A
		c.addReg8(&c.A, c.A)
	case 0x88: // ADC A,B
		c.addCarryReg8(&c.A, c.B)
	case 0x89: // ADC A,C
		c.addCarryReg8(&c.A, c.C)
	case 0x8A: // ADC A,D
		c.addCarryReg8(&c.A, c.D)
	case 0x8B: // ADC A,E
		c.addCarryReg8(&c.A, c.E)
	case 0x8C: // ADC A,H
		c.addCarryReg8(&c.A, c.H)
	case 0x8D: // ADC A,L
		c.addCarryReg8(&c.A, c.L)
	case 0x8E: // ADC A,(HL)
		c.addCarryReg8(&c.A, c.mem.Read(c.HL()))
	case 0x8F: // ADC A,A
		c.addCarryReg8(&c.A, c.A)

	// 0x9X - SUB instructions

	// 0xAX - AND, XOR instructions
	case 0xAF: // XOR A, reset A
		c.A ^= c.A
		c.F = ZERO
	// 0xBX - OR, CP instructions

	case 0xC3: // JP nn
		low := c.mem.Read(c.PC)
		high := c.mem.Read(c.PC + 1)

		c.PC = (uint16(high) << 8) | uint16(low)
	default:
		log.Fatalf("opcode unhandle %04X\n", opcode)
	}
	slog.Debug(fmt.Sprintf("opcode: 0x%04X, PC: 0x%04X  A: 0x%02X  B: 0x%02X  F: 0x%02X", opcode, c.PC, c.A, c.B, c.F))
}

func (c *CPU) addCarryReg8(reg *byte, term byte) {
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

func (c *CPU) addReg8(reg *byte, term byte) {
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

func (c *CPU) jr() {
	s8 := c.mem.Read(c.PC)
	if s8&0x80 == 0 {
		c.PC += uint16(s8 & 0x7F)
	} else {
		c.PC -= uint16(s8 & 0x7F)
	}
}

func (c *CPU) inc(reg *byte) {
	oldReg := *reg
	(*reg)++
	c.F &= 0x1F
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
	if *reg == 0 {
		c.F |= ZERO
	}

	c.F |= SUB
	if old&0x0F == 0 {
		c.F |= HALFCARRY
	}
}

func (c *CPU) Step() {
	c.Execute(c.Fetch())
}
