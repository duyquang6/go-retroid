package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/duyquang6/gboy/cpu"
	"github.com/duyquang6/gboy/mmu"
)

type State struct {
	PC  uint16      `json:"pc"`
	SP  uint16      `json:"sp"`
	A   byte        `json:"a"`
	B   byte        `json:"b"`
	C   byte        `json:"c"`
	D   byte        `json:"d"`
	E   byte        `json:"e"`
	F   byte        `json:"f"`
	H   byte        `json:"h"`
	L   byte        `json:"l"`
	IME byte        `json:"ime"`
	IE  byte        `json:"ie"`
	Ram [][2]uint16 `json:"ram"`
}

type SM83Test struct {
	Name    string `json:"name"`
	Initial State  `json:"initial"`
	Final   State  `json:"final"`
	// don't care
	// Cycles  [][]interface{} `json:"cycles"`
}

func TestSM83(t *testing.T) {
	// Get all .json files from testdata/sm83/v1
	files, err := filepath.Glob("testdata/sm83/v1/08.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		bytesData, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var sm83Tests []SM83Test
		if err := json.Unmarshal(bytesData, &sm83Tests); err != nil {
			t.Fatal(err)
		}

		for _, sm83Test := range sm83Tests {
			t.Run(fmt.Sprintf("file_%s__tc_%s", filepath.Base(file), sm83Test.Name), func(t *testing.T) {
				mem, cpu := setup(t, sm83Test.Initial)

				opcode := cpu.Fetch()
				cpu.Execute(opcode)

				if cpu.PC != sm83Test.Final.PC {
					t.Errorf("PC = %04X, want %04X", cpu.PC, sm83Test.Final.PC)
				}
				if cpu.SP != sm83Test.Final.SP {
					t.Errorf("SP = %04X, want %04X", cpu.SP, sm83Test.Final.SP)
				}
				if cpu.A != sm83Test.Final.A {
					t.Errorf("A = %02X, want %02X", cpu.A, sm83Test.Final.A)
				}

				for _, ram := range sm83Test.Final.Ram {
					got := mem.Read(uint16(ram[0]))
					if got != byte(ram[1]) {
						t.Errorf("RAM[%04X] = %02X, want %02X", ram[0], got, ram[1])
					}
				}
			})
		}

	}
}

func setup(t *testing.T, initState State) (*mmu.Memory, *cpu.CPU) {
	mem := mmu.New()
	cpu := cpu.New(mem)

	cpu.PC = initState.PC
	cpu.SP = initState.SP
	cpu.A = initState.A
	cpu.B = initState.B
	cpu.C = initState.C
	cpu.D = initState.D
	cpu.E = initState.E
	cpu.F = initState.F
	cpu.H = initState.H
	cpu.L = initState.L
	cpu.IME = initState.IME != 0
	mem.Write(0xFFFF, initState.IE)

	for _, ram := range initState.Ram {
		mem.Write(ram[0], byte(ram[1]))
	}

	return mem, cpu
}
