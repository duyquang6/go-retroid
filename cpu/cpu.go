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
		msb := c.A & 0xF0
		c.A <<= 1

		c.F = 0
		if msb == 0x10 {
			c.F |= FLAG_CARRY
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

	default:
		log.Fatalf("opcode unhandled %04X\n", opcode)
	}
	slog.Debug(fmt.Sprintf("opcode: 0x%04X, PC: 0x%04X  A: 0x%02X  B: 0x%02X  F: 0x%02X", opcode, c.PC, c.A, c.B, c.F))
}
