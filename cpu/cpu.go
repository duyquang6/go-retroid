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

func (c *CPU) Execute(opcode byte) {
	switch opcode {
	// 8 bit instruction
	case 0x00: // NOP, do nothing
	case 0x3E: // LD A,nn
		c.A = c.mem.Read(c.PC)
		c.PC++
	case 0x06: // LD B,nn
		nn := c.mem.Read(c.PC)
		c.B = nn
		c.PC++
	case 0x80:
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

func (c *CPU) Step() {
	c.Execute(c.Fetch())
}
