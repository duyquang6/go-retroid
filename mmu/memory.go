package mmu

type Memory struct {
	// 64KB memory
	data [0x10000]byte
}

func New() *Memory {
	return &Memory{}
}

func (m Memory) Read(address uint16) byte {
	return m.data[address]
}

func (m *Memory) Write(address uint16, payload byte) {
	m.data[address] = payload
}

func (m *Memory) WriteBytes(address uint16, payload []byte) {
	copy(m.data[address:address+uint16(len(payload))], payload)
}
