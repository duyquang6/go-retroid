package mmu

import (
	"fmt"
	"testing"
)

func TestMemory_Write(t *testing.T) {
	mem := Memory{}

	mem.Write(0, 1)

	fmt.Println(mem.Read(0))
}
