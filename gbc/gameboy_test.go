package gbc_test

import (
	"log/slog"
	"os"
	"testing"

	"github.com/duyquang6/gboy/gbc"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})))
}

func Test_LoadSimpleROM(t *testing.T) {
	testROMs := [][]byte{
		// Test ROM: [NOP, NOP, NOP]
		{0x00, 0x00, 0x00},
		// Test ROM: [LD B, 0x05, ADD A,B, NOP]
		{0x06, 0x05, 0x80, 0x00},
		// (LD A,5; ADD A,B; XOR A; JP 0x0100).
		{0x3E, 0x05, 0x80, 0xAF, 0xC3, 0x00, 0x01},
	}

	for _, rom := range testROMs {
		gb := gbc.NewGameBoy()
		gb.LoadROM(rom)
		gb.Run()
	}
}
