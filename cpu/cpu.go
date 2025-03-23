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
	case 0x27: // TODO: DAA
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
	case 0x2F: // TODO: CPL

	// 0x3X
	case 0x3E: // LD A,nn
		c.ldXNN(&c.A)
	case 0x80: // ADD A, B
		oldA := c.A
		sum := uint16(c.A) + uint16(c.B)
		c.A = byte(sum & 0xFF)

		c.F = 0
		if (oldA&0x0F)+(c.B&0x0F) > 0x0F {
			c.F |= HALFCARRY
		}
		if c.A == 0 {
			c.F |= ZERO
		}
		if sum > 0xFF {
			c.F |= CARRY
		}
	case 0xAF: // XOR A, reset A
		c.A ^= c.A
		c.F = ZERO
	case 0xC3: // JP nn
		low := c.mem.Read(c.PC)
		high := c.mem.Read(c.PC + 1)

		c.PC = (uint16(high) << 8) | uint16(low)
	default:
		log.Fatalf("opcode unhandle %04X\n", opcode)
	}
	slog.Debug(fmt.Sprintf("opcode: 0x%04X, PC: 0x%04X  A: 0x%02X  B: 0x%02X  F: 0x%02X", opcode, c.PC, c.A, c.B, c.F))
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
