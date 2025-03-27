package cpu

import (
	"fmt"
	"log"
	"log/slog"

	"github.com/duyquang6/gboy/mmu"
)

type CPU struct {
	// Accumulator Register (Fast processing ALU)
	A byte
	// Flag register: 0x80 = Zero, 0x40 = Subtract, 0x20 = Half Carry, 0x10 = carry
	F byte

	// General purpose
	B, C, D, E, H, L byte

	PC, SP uint16

	// interupt master enable
	IME bool

	mem *mmu.Memory

	stopped bool
}

func New(mem *mmu.Memory) *CPU {
	// follow Gameboy BIOS spec
	return &CPU{
		A:       0x01,   // Accumulator
		F:       0xB0,   // Flags
		B:       0x00,   // General-purpose register B
		C:       0x13,   // General-purpose register C
		D:       0x00,   // General-purpose register D
		E:       0xD8,   // General-purpose register E
		H:       0x01,   // General-purpose register H
		L:       0x4D,   // General-purpose register L
		PC:      0x0100, // Program Counter starts at 0x0100
		SP:      0xFFFE, // Stack Pointer starts at 0xFFFE
		mem:     mem,    // Memory reference
		stopped: false,  // CPU is not stopped initially
	}
}

func (c *CPU) Memory() *mmu.Memory {
	return c.mem
}

func (c *CPU) Fetch() byte {
	opcode := c.mem.Read(c.PC)
	c.PC++

	return opcode
}

func (c *CPU) Step() {
	c.Execute(c.Fetch())
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
		msb := c.A & 0x80
		c.A <<= 1

		c.F = 0
		if msb != 0 {
			c.F |= FLAG_CARRY
			c.A |= 0x01
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
			c.F |= FLAG_HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= FLAG_CARRY
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
			c.F |= FLAG_CARRY
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
		if c.F&FLAG_CARRY > 0 {
			c.A |= 0x01
		}

		c.F = 0
		if oldA&0x80 > 0 {
			c.F = FLAG_CARRY
		}
	case 0x18: // JR s8
		c.jr()
	case 0x19: // ADD HL, DE
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.DE())
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(c.DE()&0x00FF) > 0x00FF {
			c.F |= FLAG_HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= FLAG_CARRY
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
		if c.F&FLAG_CARRY > 0 {
			c.A |= 0x80
		}
		c.F = 0
		if oldA&0x01 > 0 {
			c.F = FLAG_CARRY
		}
	// 0x2X
	case 0x20: // JR NZ, s8
		if c.F&FLAG_ZERO == 0 {
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
		if c.F&FLAG_SUBTRACT == 0 {
			// Addition
			if (c.A&0x0F) > 9 || (c.F&FLAG_HALFCARRY) != 0 {
				c.A += 0x06
			}
			if c.A > 0x99 || (c.F&FLAG_CARRY) != 0 {
				c.A += 0x60
				c.F |= FLAG_CARRY
			}
		} else {
			if (c.F & FLAG_HALFCARRY) != 0 {
				c.A -= 0x06
			}
			if (c.F & FLAG_CARRY) != 0 {
				c.A -= 0x60
			}
		}
		// Reset H to 0
		c.F &= ^FLAG_HALFCARRY

		if c.A == 0 {
			c.F |= FLAG_ZERO
		} else {
			c.F &= ^FLAG_ZERO
		}
	case 0x28: // JR Z,s8
		if c.F&FLAG_ZERO != 0 {
			c.jr()
		}
	case 0x29: // ADD HL,HL
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.HL())
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(old&0x00FF) > 0x00FF {
			c.F |= FLAG_HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= FLAG_CARRY
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
		c.F |= FLAG_HALFCARRY | FLAG_SUBTRACT

	// 0x3X
	case 0x30: // JR NC, s8
		if (c.F & FLAG_CARRY) != 0 {
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
			c.F |= FLAG_ZERO
		}
		if old&0x0F == 0x0F {
			c.F |= FLAG_HALFCARRY
		}
	case 0x35: // DEC (HL)
		val := c.mem.Read(c.HL())
		old := val
		val--
		c.mem.Write(c.HL(), val)

		if val == 0 {
			c.F |= FLAG_ZERO
		}
		c.F |= FLAG_SUBTRACT
		if old&0x0F == 0 {
			c.F |= FLAG_HALFCARRY
		}
	case 0x36: // LD (HL),d8
		val := c.mem.Read(c.PC)
		c.mem.Write(c.HL(), val)
		c.PC++
	case 0x37: // SCF
		c.F = (c.F & FLAG_ZERO) | FLAG_CARRY
	case 0x38: // JR C,s8
		if c.F&FLAG_CARRY != 0 {
			c.jr()
		}
	case 0x39: // ADD HL,SP
		old := c.HL()
		sum := uint32(c.HL()) + uint32(c.SP)
		c.WriteHL(uint16(sum & 0xFFFF))
		c.F &= 0x80
		if (old&0x00FF)+(uint16(c.SP)&0x00FF) > 0x00FF {
			c.F |= FLAG_HALFCARRY
		}
		if sum > 0xFFFF {
			c.F |= FLAG_CARRY
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
		c.F = (c.F ^ FLAG_CARRY) & (FLAG_ZERO | FLAG_CARRY)

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

	// 0x4X - Load instructions C
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
		c.add(&c.A, c.B)
	case 0x81: // ADD A,C
		c.add(&c.A, c.C)
	case 0x82: // ADD A,D
		c.add(&c.A, c.D)
	case 0x83: // ADD A,E
		c.add(&c.A, c.E)
	case 0x84: // ADD A,H
		c.add(&c.A, c.H)
	case 0x85: // ADD A,L
		c.add(&c.A, c.L)
	case 0x86: // ADD A,(HL)
		c.add(&c.A, c.mem.Read(c.HL()))
	case 0x87: // ADD A,A
		c.add(&c.A, c.A)
	case 0x88: // ADC A,B
		c.addCarry(&c.A, c.B)
	case 0x89: // ADC A,C
		c.addCarry(&c.A, c.C)
	case 0x8A: // ADC A,D
		c.addCarry(&c.A, c.D)
	case 0x8B: // ADC A,E
		c.addCarry(&c.A, c.E)
	case 0x8C: // ADC A,H
		c.addCarry(&c.A, c.H)
	case 0x8D: // ADC A,L
		c.addCarry(&c.A, c.L)
	case 0x8E: // ADC A,(HL)
		c.addCarry(&c.A, c.mem.Read(c.HL()))
	case 0x8F: // ADC A,A
		c.addCarry(&c.A, c.A)

	// 0x9X - SUB instructions
	case 0x90: // SUB B ~ SUB A, B
		c.sub(&c.A, c.B)
	case 0x91: // SUB C ~ SUB A, C
		c.sub(&c.A, c.C)
	case 0x92: // SUB D ~ SUB A, D
		c.sub(&c.A, c.D)
	case 0x93: // SUB E ~ SUB A, E
		c.sub(&c.A, c.E)
	case 0x94: // SUB H ~ SUB A, H
		c.sub(&c.A, c.H)
	case 0x95: // SUB L ~ SUB A, L
		c.sub(&c.A, c.L)
	case 0x96: // SUB (HL) ~ SUB A, (HL)
		c.sub(&c.A, c.mem.Read(c.HL()))
	case 0x97: // SUB A ~ SUB A, A
		c.sub(&c.A, c.A)
	case 0x98: // SBC A, B
		c.subCarry(&c.A, c.B)
	case 0x99: // SBC A,C
		c.subCarry(&c.A, c.C)
	case 0x9A: // SBC A,D
		c.subCarry(&c.A, c.D)
	case 0x9B: // SBC A,E
		c.subCarry(&c.A, c.E)
	case 0x9C: // SBC A,H
		c.subCarry(&c.A, c.H)
	case 0x9D: // SBC A,L
		c.subCarry(&c.A, c.L)
	case 0x9E: // SBC A,(HL)
		c.subCarry(&c.A, c.mem.Read(c.HL()))
	case 0x9F: // SBC A,A
		c.subCarry(&c.A, c.A)

	// 0xAX - AND, XOR instructions
	case 0xA0: // AND B
		c.and(&c.A, c.B)
	case 0xA1: // AND C
		c.and(&c.A, c.C)
	case 0xA2: // AND D
		c.and(&c.A, c.D)
	case 0xA3: // AND E
		c.and(&c.A, c.E)
	case 0xA4: // AND H
		c.and(&c.A, c.H)
	case 0xA5: // AND L
		c.and(&c.A, c.L)
	case 0xA6: // AND (HL)
		c.and(&c.A, c.mem.Read(c.HL()))
	case 0xA7: // AND A
		c.and(&c.A, c.A)
	case 0xA8: // XOR B
		c.xor(&c.A, c.B)
	case 0xA9: // XOR C
		c.xor(&c.A, c.C)
	case 0xAA: // XOR D
		c.xor(&c.A, c.D)
	case 0xAB: // XOR E
		c.xor(&c.A, c.E)
	case 0xAC: // XOR H
		c.xor(&c.A, c.H)
	case 0xAD: // XOR L
		c.xor(&c.A, c.L)
	case 0xAE: // XOR (HL)
		c.xor(&c.A, c.mem.Read(c.HL()))
	case 0xAF: // XOR A
		c.xor(&c.A, c.A)

	// 0xBX - OR, CP instructions
	case 0xB0: // OR B
		c.or(&c.A, c.B)
	case 0xB1: // OR C
		c.or(&c.A, c.C)
	case 0xB2: // OR D
		c.or(&c.A, c.D)
	case 0xB3: // OR E
		c.or(&c.A, c.E)
	case 0xB4: // OR H
		c.or(&c.A, c.H)
	case 0xB5: // OR L
		c.or(&c.A, c.L)
	case 0xB6: // OR (HL)
		c.or(&c.A, c.mem.Read(c.HL()))
	case 0xB7: // OR A
		c.or(&c.A, c.A)
	case 0xB8: // CP B
		c.cp(c.A, c.B)
	case 0xB9: // CP C
		c.cp(c.A, c.C)
	case 0xBA: // CP D
		c.cp(c.A, c.D)
	case 0xBB: // CP E
		c.cp(c.A, c.E)
	case 0xBC: // CP H
		c.cp(c.A, c.H)
	case 0xBD: // CP L
		c.cp(c.A, c.L)
	case 0xBE: // CP (HL)
		c.cp(c.A, c.mem.Read(c.HL()))
	case 0xBF: // CP A
		c.cp(c.A, c.A)

	// 0xCX, Jump, RET, etc,...
	case 0xC0: // RET NZ
		if c.F&FLAG_ZERO == 0 {
			c.ret()
		}
	case 0xC1: // POP BC
		low := c.mem.Read(c.SP)
		high := c.mem.Read(c.SP + 1)
		c.WriteBC(uint16(high)<<8 | uint16(low))
		c.SP += 2
	case 0xC2: // JP NZ, a16
		if c.F&FLAG_ZERO == 0 {
			c.jp()
		} else {
			c.PC++
		}
	case 0xC3: // JP a16
		c.jp()
	case 0xC4: // CALL NZ, a16
		if c.F&FLAG_ZERO == 0 {
			c.call()
		} else {
			c.PC += 2
		}
	case 0xC5: // PUSH BC
		c.SP -= 2
		c.mem.Write(c.SP, c.C)
		c.mem.Write(c.SP+1, c.B)
	case 0xC6: // ADD A, d8
		c.add(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xC7: // RST 0
		c.rst()
		c.PC = 0x0000
	case 0xC8: // RET Z
		if c.F&FLAG_ZERO != 0 {
			c.ret()
		}
	case 0xC9: // RET
		c.ret()
	case 0xCA: // JP Z, a16
		if c.F&FLAG_ZERO != 0 {
			c.jp()
		} else {
			c.PC += 2
		}
	case 0xCC: // CALL Z, a16
		if c.F&FLAG_ZERO != 0 {
			c.call()
		} else {
			c.PC += 2
		}
	case 0xCD: // CALL a16
		c.call()
	case 0xCE: // ADC A, d8
		c.addCarry(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xCF: // RST 1
		c.rst()
		c.PC = 0x0008

	// 0xDX - CALL, PUSH, SUB, etc.
	case 0xD0: // RET NC
		if c.F&FLAG_CARRY == 0 {
			c.ret()
		}
	case 0xD1: // POP DE
		low := c.mem.Read(c.SP)
		high := c.mem.Read(c.SP + 1)
		c.WriteDE(uint16(high)<<8 | uint16(low))
		c.SP += 2
	case 0xD2: // JP NC, a16
		if c.F&FLAG_CARRY == 0 {
			c.jp()
		} else {
			c.PC += 2
		}
	case 0xD3: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xD3")
	case 0xD4: // CALL NC, a16
		if c.F&FLAG_CARRY == 0 {
			c.call()
		} else {
			c.PC += 2
		}
	case 0xD5: // PUSH DE
		c.SP -= 2
		c.mem.Write(c.SP, c.E)
		c.mem.Write(c.SP+1, c.D)
	case 0xD6: // SUB d8
		c.sub(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xD7: // RST 2
		c.rst()
		c.PC = 0x0010
	case 0xD8: // RET C
		if c.F&FLAG_CARRY != 0 {
			c.ret()
		}
	case 0xD9: // RETI
		c.ret()
		c.IME = true // Enable interrupts
	case 0xDA: // JP C, a16
		if c.F&FLAG_CARRY != 0 {
			c.jp()
		} else {
			c.PC += 2
		}
	case 0xDB: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xDB")
	case 0xDC: // CALL C, a16
		if c.F&FLAG_CARRY != 0 {
			c.call()
		} else {
			c.PC += 2
		}
	case 0xDD: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xDD")
	case 0xDE: // SBC A, d8
		c.subCarry(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xDF: // RST 3
		c.rst()
		c.PC = 0x0018

	// 0xEX - LD, PUSH, etc.
	case 0xE0: // LD (a8), A
		addr := 0xFF00 + uint16(c.mem.Read(c.PC))
		c.mem.Write(addr, c.A)
		c.PC++
	case 0xE1: // POP HL
		low := c.mem.Read(c.SP)
		high := c.mem.Read(c.SP + 1)
		c.WriteHL(uint16(high)<<8 | uint16(low))
		c.SP += 2
	case 0xE2: // LD (C), A
		addr := 0xFF00 + uint16(c.C)
		c.mem.Write(addr, c.A)
	case 0xE3: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xE3")
	case 0xE4: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xE4")
	case 0xE5: // PUSH HL
		c.SP -= 2
		c.mem.Write(c.SP, c.L)
		c.mem.Write(c.SP+1, c.H)
	case 0xE6: // AND d8
		c.and(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xE7: // RST 4
		c.rst()
		c.PC = 0x0020
	case 0xE8: // ADD SP, r8
		offset := int8(c.mem.Read(c.PC))
		c.PC++
		oldSP := c.SP
		c.SP = uint16(int32(c.SP) + int32(offset))
		c.F = 0
		if (oldSP&0x0F)+(uint16(offset)&0x0F) > 0x0F {
			c.F |= FLAG_HALFCARRY
		}
		if (oldSP&0xFF)+(uint16(offset)&0xFF) > 0xFF {
			c.F |= FLAG_CARRY
		}
	case 0xE9: // JP (HL)
		c.PC = c.HL()
	case 0xEA: // LD (a16), A
		addr := uint16(c.mem.Read(c.PC)) | uint16(c.mem.Read(c.PC+1))<<8
		c.mem.Write(addr, c.A)
		c.PC += 2
	case 0xEB: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xEB")
	case 0xEC: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xEC")
	case 0xED: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xED")
	case 0xEE: // XOR d8
		c.xor(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xEF: // RST 5
		c.rst()
		c.PC = 0x0028

	// 0xFX - LD, CP, etc.
	case 0xF0: // LDH A, (a8)
		addr := 0xFF00 + uint16(c.mem.Read(c.PC))
		c.A = c.mem.Read(addr)
		c.PC++
	case 0xF1: // POP AF
		low := c.mem.Read(c.SP)
		high := c.mem.Read(c.SP + 1)
		c.A = high
		c.F = low & 0xF0
		c.SP += 2
	case 0xF2: // LD A, (C)
		addr := 0xFF00 + uint16(c.C)
		c.A = c.mem.Read(addr)
	case 0xF3: // DI
		c.IME = false // Disable interrupts
	case 0xF4: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xF4")
	case 0xF5: // PUSH AF
		c.SP -= 2
		c.mem.Write(c.SP, c.F)
		c.mem.Write(c.SP+1, c.A)
	case 0xF6: // OR d8
		c.or(&c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xF7: // RST 6
		c.rst()
		c.PC = 0x0030
	case 0xF8: // LD HL, SP+s8
		offset := int8(c.mem.Read(c.PC))
		c.PC++
		result := uint16(int32(c.SP) + int32(offset))
		c.WriteHL(result)
		c.F = 0
		if (c.SP&0x0F)+(uint16(offset)&0x0F) > 0x0F {
			c.F |= FLAG_HALFCARRY
		}
		if (c.SP&0xFF)+(uint16(offset)&0xFF) > 0xFF {
			c.F |= FLAG_CARRY
		}
	case 0xF9: // LD SP, HL
		c.SP = c.HL()
	case 0xFA: // LD A, (a16)
		addr := uint16(c.mem.Read(c.PC)) | uint16(c.mem.Read(c.PC+1))<<8
		c.A = c.mem.Read(addr)
		c.PC += 2
	case 0xFB: // EI
		c.IME = true // Enable interrupts
	case 0xFC: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xFC")
	case 0xFD: // Unused (illegal opcode)
		log.Fatalf("Illegal opcode: 0xFD")
	case 0xFE: // CP d8
		c.cp(c.A, c.mem.Read(c.PC))
		c.PC++
	case 0xFF: // RST 7
		c.rst()
		c.PC = 0x0038

	case 0xCB:
		c.handleCBx()
	default:
		log.Fatalf("opcode unhandled %04X\n", opcode)
	}
	slog.Debug(fmt.Sprintf("opcode: 0x%04X, PC: 0x%04X  A: 0x%02X  B: 0x%02X  F: 0x%02X", opcode, c.PC, c.A, c.B, c.F))
}

func (c *CPU) handleCBx() {
	opcode := c.mem.Read(c.PC)
	c.PC++

	switch opcode {
	case 0x00: // RLC B
		c.rlc(&c.B)
	case 0x01: // RLC C
		c.rlc(&c.C)
	case 0x02: // RLC D
		c.rlc(&c.D)
	case 0x03: // RLC E
		c.rlc(&c.E)
	case 0x04: // RLC H
		c.rlc(&c.H)
	case 0x05: // RLC L
		c.rlc(&c.L)
	case 0x06: // RLC (HL)
		val := c.mem.Read(c.HL())
		c.rlc(&val)
		c.mem.Write(c.HL(), val)
	case 0x07: // RLC A
		c.rlc(&c.A)
	case 0x08: // RRC B
		c.rrc(&c.B)
	case 0x09: // RRC C
		c.rrc(&c.C)
	case 0x0A: // RRC D
		c.rrc(&c.D)
	case 0x0B: // RRC E
		c.rrc(&c.E)
	case 0x0C: // RRC H
		c.rrc(&c.H)
	case 0x0D: // RRC L
		c.rrc(&c.L)
	case 0x0E: // RRC (HL)
		val := c.mem.Read(c.HL())
		c.rrc(&val)
		c.mem.Write(c.HL(), val)
	case 0x0F: // RRC A
		c.rrc(&c.A)
	case 0x10: // RL B
		c.rl(&c.B)
	case 0x11: // RL C
		c.rl(&c.C)
	case 0x12: // RL D
		c.rl(&c.D)
	case 0x13: // RL E
		c.rl(&c.E)
	case 0x14: // RL H
		c.rl(&c.H)
	case 0x15: // RL L
		c.rl(&c.L)
	case 0x16: // RL (HL)
		val := c.mem.Read(c.HL())
		c.rl(&val)
		c.mem.Write(c.HL(), val)
	case 0x17: // RL A
		c.rl(&c.A)
	case 0x18: // RR B
		c.rr(&c.B)
	case 0x19: // RR C
		c.rr(&c.C)
	case 0x1A: // RR D
		c.rr(&c.D)
	case 0x1B: // RR E
		c.rr(&c.E)
	case 0x1C: // RR H
		c.rr(&c.H)
	case 0x1D: // RR L
		c.rr(&c.L)
	case 0x1E: // RR (HL)
		val := c.mem.Read(c.HL())
		c.rr(&val)
		c.mem.Write(c.HL(), val)
	case 0x1F: // RR A
		c.rr(&c.A)

	case 0x20: // SLA B
		c.sla(&c.B)
	case 0x21: // SLA C
		c.sla(&c.C)
	case 0x22: // SLA D
		c.sla(&c.D)
	case 0x23: // SLA E
		c.sla(&c.E)
	case 0x24: // SLA H
		c.sla(&c.H)
	case 0x25: // SLA L
		c.sla(&c.L)
	case 0x26: // SLA (HL)
		val := c.mem.Read(c.HL())
		c.sla(&val)
		c.mem.Write(c.HL(), val)
	case 0x27: // SLA A
		c.sla(&c.A)
	case 0x28: // SRA B
		c.sra(&c.B)
	case 0x29: // SRA C
		c.sra(&c.C)
	case 0x2A: // SRA D
		c.sra(&c.D)
	case 0x2B: // SRA E
		c.sra(&c.E)
	case 0x2C: // SRA H
		c.sra(&c.H)
	case 0x2D: // SRA L
		c.sra(&c.L)
	case 0x2E: // SRA (HL)
		val := c.mem.Read(c.HL())
		c.sra(&val)
		c.mem.Write(c.HL(), val)
	case 0x2F: // SRA A
		c.sra(&c.A)

	case 0x30: // SWAP B
		c.swap(&c.B)
	case 0x31: // SWAP C
		c.swap(&c.C)
	case 0x32: // SWAP D
		c.swap(&c.D)
	case 0x33: // SWAP E
		c.swap(&c.E)
	case 0x34: // SWAP H
		c.swap(&c.H)
	case 0x35: // SWAP L
		c.swap(&c.L)
	case 0x36: // SWAP (HL)
		val := c.mem.Read(c.HL())
		c.swap(&val)
		c.mem.Write(c.HL(), val)
	case 0x37: // SWAP A
		c.swap(&c.A)
	case 0x38: // SRL B
		c.srl(&c.B)
	case 0x39: // SRL C
		c.srl(&c.C)
	case 0x3A: // SRL D
		c.srl(&c.D)
	case 0x3B: // SRL E
		c.srl(&c.E)
	case 0x3C: // SRL H
		c.srl(&c.H)
	case 0x3D: // SRL L
		c.srl(&c.L)
	case 0x3E: // SRL (HL)
		val := c.mem.Read(c.HL())
		c.srl(&val)
		c.mem.Write(c.HL(), val)
	case 0x3F: // SRL A
		c.srl(&c.A)

	// BIT b,r instructions - Test bit b in register r
	case 0x40: // BIT 0,B
		c.bit(0, c.B)
	case 0x41: // BIT 0,C
		c.bit(0, c.C)
	case 0x42: // BIT 0,D
		c.bit(0, c.D)
	case 0x43: // BIT 0,E
		c.bit(0, c.E)
	case 0x44: // BIT 0,H
		c.bit(0, c.H)
	case 0x45: // BIT 0,L
		c.bit(0, c.L)
	case 0x46: // BIT 0,(HL)
		c.bit(0, c.mem.Read(c.HL()))
	case 0x47: // BIT 0,A
		c.bit(0, c.A)

	case 0x48: // BIT 1,B
		c.bit(1, c.B)
	case 0x49: // BIT 1,C
		c.bit(1, c.C)
	case 0x4A: // BIT 1,D
		c.bit(1, c.D)
	case 0x4B: // BIT 1,E
		c.bit(1, c.E)
	case 0x4C: // BIT 1,H
		c.bit(1, c.H)
	case 0x4D: // BIT 1,L
		c.bit(1, c.L)
	case 0x4E: // BIT 1,(HL)
		c.bit(1, c.mem.Read(c.HL()))
	case 0x4F: // BIT 1,A
		c.bit(1, c.A)

	case 0x50: // BIT 2,B
		c.bit(2, c.B)
	case 0x51: // BIT 2,C
		c.bit(2, c.C)
	case 0x52: // BIT 2,D
		c.bit(2, c.D)
	case 0x53: // BIT 2,E
		c.bit(2, c.E)
	case 0x54: // BIT 2,H
		c.bit(2, c.H)
	case 0x55: // BIT 2,L
		c.bit(2, c.L)
	case 0x56: // BIT 2,(HL)
		c.bit(2, c.mem.Read(c.HL()))
	case 0x57: // BIT 2,A
		c.bit(2, c.A)

	case 0x58: // BIT 3,B
		c.bit(3, c.B)
	case 0x59: // BIT 3,C
		c.bit(3, c.C)
	case 0x5A: // BIT 3,D
		c.bit(3, c.D)
	case 0x5B: // BIT 3,E
		c.bit(3, c.E)
	case 0x5C: // BIT 3,H
		c.bit(3, c.H)
	case 0x5D: // BIT 3,L
		c.bit(3, c.L)
	case 0x5E: // BIT 3,(HL)
		c.bit(3, c.mem.Read(c.HL()))
	case 0x5F: // BIT 3,A
		c.bit(3, c.A)

	case 0x60: // BIT 4,B
		c.bit(4, c.B)
	case 0x61: // BIT 4,C
		c.bit(4, c.C)
	case 0x62: // BIT 4,D
		c.bit(4, c.D)
	case 0x63: // BIT 4,E
		c.bit(4, c.E)
	case 0x64: // BIT 4,H
		c.bit(4, c.H)
	case 0x65: // BIT 4,L
		c.bit(4, c.L)
	case 0x66: // BIT 4,(HL)
		c.bit(4, c.mem.Read(c.HL()))
	case 0x67: // BIT 4,A
		c.bit(4, c.A)

	case 0x68: // BIT 5,B
		c.bit(5, c.B)
	case 0x69: // BIT 5,C
		c.bit(5, c.C)
	case 0x6A: // BIT 5,D
		c.bit(5, c.D)
	case 0x6B: // BIT 5,E
		c.bit(5, c.E)
	case 0x6C: // BIT 5,H
		c.bit(5, c.H)
	case 0x6D: // BIT 5,L
		c.bit(5, c.L)
	case 0x6E: // BIT 5,(HL)
		c.bit(5, c.mem.Read(c.HL()))
	case 0x6F: // BIT 5,A
		c.bit(5, c.A)

	case 0x70: // BIT 6,B
		c.bit(6, c.B)
	case 0x71: // BIT 6,C
		c.bit(6, c.C)
	case 0x72: // BIT 6,D
		c.bit(6, c.D)
	case 0x73: // BIT 6,E
		c.bit(6, c.E)
	case 0x74: // BIT 6,H
		c.bit(6, c.H)
	case 0x75: // BIT 6,L
		c.bit(6, c.L)
	case 0x76: // BIT 6,(HL)
		c.bit(6, c.mem.Read(c.HL()))
	case 0x77: // BIT 6,A
		c.bit(6, c.A)

	case 0x78: // BIT 7,B
		c.bit(7, c.B)
	case 0x79: // BIT 7,C
		c.bit(7, c.C)
	case 0x7A: // BIT 7,D
		c.bit(7, c.D)
	case 0x7B: // BIT 7,E
		c.bit(7, c.E)
	case 0x7C: // BIT 7,H
		c.bit(7, c.H)
	case 0x7D: // BIT 7,L
		c.bit(7, c.L)
	case 0x7E: // BIT 7,(HL)
		c.bit(7, c.mem.Read(c.HL()))
	case 0x7F: // BIT 7,A
		c.bit(7, c.A)

	// RES instructions (0x80-0xBF)
	case 0x80: // RES 0,B
		c.res(0, &c.B)
	case 0x81: // RES 0,C
		c.res(0, &c.C)
	case 0x82: // RES 0,D
		c.res(0, &c.D)
	case 0x83: // RES 0,E
		c.res(0, &c.E)
	case 0x84: // RES 0,H
		c.res(0, &c.H)
	case 0x85: // RES 0,L
		c.res(0, &c.L)
	case 0x86: // RES 0,(HL)
		val := c.mem.Read(c.HL())
		c.res(0, &val)
		c.mem.Write(c.HL(), val)
	case 0x87: // RES 0,A
		c.res(0, &c.A)
	case 0x88: // RES 1,B
		c.res(1, &c.B)
	case 0x89: // RES 1,C
		c.res(1, &c.C)
	case 0x8A: // RES 1,D
		c.res(1, &c.D)
	case 0x8B: // RES 1,E
		c.res(1, &c.E)
	case 0x8C: // RES 1,H
		c.res(1, &c.H)
	case 0x8D: // RES 1,L
		c.res(1, &c.L)
	case 0x8E: // RES 1,(HL)
		val := c.mem.Read(c.HL())
		c.res(1, &val)
		c.mem.Write(c.HL(), val)
	case 0x8F: // RES 1,A
		c.res(1, &c.A)
	case 0x90: // RES 2,B
		c.res(2, &c.B)
	case 0x91: // RES 2,C
		c.res(2, &c.C)
	case 0x92: // RES 2,D
		c.res(2, &c.D)
	case 0x93: // RES 2,E
		c.res(2, &c.E)
	case 0x94: // RES 2,H
		c.res(2, &c.H)
	case 0x95: // RES 2,L
		c.res(2, &c.L)
	case 0x96: // RES 2,(HL)
		val := c.mem.Read(c.HL())
		c.res(2, &val)
		c.mem.Write(c.HL(), val)
	case 0x97: // RES 2,A
		c.res(2, &c.A)
	case 0x98: // RES 3,B
		c.res(3, &c.B)
	case 0x99: // RES 3,C
		c.res(3, &c.C)
	case 0x9A: // RES 3,D
		c.res(3, &c.D)
	case 0x9B: // RES 3,E
		c.res(3, &c.E)
	case 0x9C: // RES 3,H
		c.res(3, &c.H)
	case 0x9D: // RES 3,L
		c.res(3, &c.L)
	case 0x9E: // RES 3,(HL)
		val := c.mem.Read(c.HL())
		c.res(3, &val)
		c.mem.Write(c.HL(), val)
	case 0x9F: // RES 3,A
		c.res(3, &c.A)
	case 0xA0: // RES 4,B
		c.res(4, &c.B)
	case 0xA1: // RES 4,C
		c.res(4, &c.C)
	case 0xA2: // RES 4,D
		c.res(4, &c.D)
	case 0xA3: // RES 4,E
		c.res(4, &c.E)
	case 0xA4: // RES 4,H
		c.res(4, &c.H)
	case 0xA5: // RES 4,L
		c.res(4, &c.L)
	case 0xA6: // RES 4,(HL)
		val := c.mem.Read(c.HL())
		c.res(4, &val)
		c.mem.Write(c.HL(), val)
	case 0xA7: // RES 4,A
		c.res(4, &c.A)
	case 0xA8: // RES 5,B
		c.res(5, &c.B)
	case 0xA9: // RES 5,C
		c.res(5, &c.C)
	case 0xAA: // RES 5,D
		c.res(5, &c.D)
	case 0xAB: // RES 5,E
		c.res(5, &c.E)
	case 0xAC: // RES 5,H
		c.res(5, &c.H)
	case 0xAD: // RES 5,L
		c.res(5, &c.L)
	case 0xAE: // RES 5,(HL)
		val := c.mem.Read(c.HL())
		c.res(5, &val)
		c.mem.Write(c.HL(), val)
	case 0xAF: // RES 5,A
		c.res(5, &c.A)
	case 0xB0: // RES 6,B
		c.res(6, &c.B)
	case 0xB1: // RES 6,C
		c.res(6, &c.C)
	case 0xB2: // RES 6,D
		c.res(6, &c.D)
	case 0xB3: // RES 6,E
		c.res(6, &c.E)
	case 0xB4: // RES 6,H
		c.res(6, &c.H)
	case 0xB5: // RES 6,L
		c.res(6, &c.L)
	case 0xB6: // RES 6,(HL)
		val := c.mem.Read(c.HL())
		c.res(6, &val)
		c.mem.Write(c.HL(), val)
	case 0xB7: // RES 6,A
		c.res(6, &c.A)
	case 0xB8: // RES 7,B
		c.res(7, &c.B)
	case 0xB9: // RES 7,C
		c.res(7, &c.C)
	case 0xBA: // RES 7,D
		c.res(7, &c.D)
	case 0xBB: // RES 7,E
		c.res(7, &c.E)
	case 0xBC: // RES 7,H
		c.res(7, &c.H)
	case 0xBD: // RES 7,L
		c.res(7, &c.L)
	case 0xBE: // RES 7,(HL)
		val := c.mem.Read(c.HL())
		c.res(7, &val)
		c.mem.Write(c.HL(), val)
	case 0xBF: // RES 7,A
		c.res(7, &c.A)

	// SET instructions (0xC0-0xFF)
	case 0xC0: // SET 0,B
		c.set(0, &c.B)
	case 0xC1: // SET 0,C
		c.set(0, &c.C)
	case 0xC2: // SET 0,D
		c.set(0, &c.D)
	case 0xC3: // SET 0,E
		c.set(0, &c.E)
	case 0xC4: // SET 0,H
		c.set(0, &c.H)
	case 0xC5: // SET 0,L
		c.set(0, &c.L)
	case 0xC6: // SET 0,(HL)
		val := c.mem.Read(c.HL())
		c.set(0, &val)
		c.mem.Write(c.HL(), val)
	case 0xC7: // SET 0,A
		c.set(0, &c.A)
	case 0xC8: // SET 1,B
		c.set(1, &c.B)
	case 0xC9: // SET 1,C
		c.set(1, &c.C)
	case 0xCA: // SET 1,D
		c.set(1, &c.D)
	case 0xCB: // SET 1,E
		c.set(1, &c.E)
	case 0xCC: // SET 1,H
		c.set(1, &c.H)
	case 0xCD: // SET 1,L
		c.set(1, &c.L)
	case 0xCE: // SET 1,(HL)
		val := c.mem.Read(c.HL())
		c.set(1, &val)
		c.mem.Write(c.HL(), val)
	case 0xCF: // SET 1,A
		c.set(1, &c.A)
	case 0xD0: // SET 2,B
		c.set(2, &c.B)
	case 0xD1: // SET 2,C
		c.set(2, &c.C)
	case 0xD2: // SET 2,D
		c.set(2, &c.D)
	case 0xD3: // SET 2,E
		c.set(2, &c.E)
	case 0xD4: // SET 2,H
		c.set(2, &c.H)
	case 0xD5: // SET 2,L
		c.set(2, &c.L)
	case 0xD6: // SET 2,(HL)
		val := c.mem.Read(c.HL())
		c.set(2, &val)
		c.mem.Write(c.HL(), val)
	case 0xD7: // SET 2,A
		c.set(2, &c.A)
	case 0xD8: // SET 3,B
		c.set(3, &c.B)
	case 0xD9: // SET 3,C
		c.set(3, &c.C)
	case 0xDA: // SET 3,D
		c.set(3, &c.D)
	case 0xDB: // SET 3,E
		c.set(3, &c.E)
	case 0xDC: // SET 3,H
		c.set(3, &c.H)
	case 0xDD: // SET 3,L
		c.set(3, &c.L)
	case 0xDE: // SET 3,(HL)
		val := c.mem.Read(c.HL())
		c.set(3, &val)
		c.mem.Write(c.HL(), val)
	case 0xDF: // SET 3,A
		c.set(3, &c.A)
	case 0xE0: // SET 4,B
		c.set(4, &c.B)
	case 0xE1: // SET 4,C
		c.set(4, &c.C)
	case 0xE2: // SET 4,D
		c.set(4, &c.D)
	case 0xE3: // SET 4,E
		c.set(4, &c.E)
	case 0xE4: // SET 4,H
		c.set(4, &c.H)
	case 0xE5: // SET 4,L
		c.set(4, &c.L)
	case 0xE6: // SET 4,(HL)
		val := c.mem.Read(c.HL())
		c.set(4, &val)
		c.mem.Write(c.HL(), val)
	case 0xE7: // SET 4,A
		c.set(4, &c.A)
	case 0xE8: // SET 5,B
		c.set(5, &c.B)
	case 0xE9: // SET 5,C
		c.set(5, &c.C)
	case 0xEA: // SET 5,D
		c.set(5, &c.D)
	case 0xEB: // SET 5,E
		c.set(5, &c.E)
	case 0xEC: // SET 5,H
		c.set(5, &c.H)
	case 0xED: // SET 5,L
		c.set(5, &c.L)
	case 0xEE: // SET 5,(HL)
		val := c.mem.Read(c.HL())
		c.set(5, &val)
		c.mem.Write(c.HL(), val)
	case 0xEF: // SET 5,A
		c.set(5, &c.A)
	case 0xF0: // SET 6,B
		c.set(6, &c.B)
	case 0xF1: // SET 6,C
		c.set(6, &c.C)
	case 0xF2: // SET 6,D
		c.set(6, &c.D)
	case 0xF3: // SET 6,E
		c.set(6, &c.E)
	case 0xF4: // SET 6,H
		c.set(6, &c.H)
	case 0xF5: // SET 6,L
		c.set(6, &c.L)
	case 0xF6: // SET 6,(HL)
		val := c.mem.Read(c.HL())
		c.set(6, &val)
		c.mem.Write(c.HL(), val)
	case 0xF7: // SET 6,A
		c.set(6, &c.A)
	case 0xF8: // SET 7,B
		c.set(7, &c.B)
	case 0xF9: // SET 7,C
		c.set(7, &c.C)
	case 0xFA: // SET 7,D
		c.set(7, &c.D)
	case 0xFB: // SET 7,E
		c.set(7, &c.E)
	case 0xFC: // SET 7,H
		c.set(7, &c.H)
	case 0xFD: // SET 7,L
		c.set(7, &c.L)
	case 0xFE: // SET 7,(HL)
		val := c.mem.Read(c.HL())
		c.set(7, &val)
		c.mem.Write(c.HL(), val)
	case 0xFF: // SET 7,A
		c.set(7, &c.A)

	default:
		log.Fatalf("Unhandled CB opcode: 0x%02X", opcode)
	}
}
