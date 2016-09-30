package core

// nn = int16
// n = int8
var instructionSetDeclaration = map[int]*instruction{
	0x00: &instruction{name: "NOP", ticks: 4, length: 1, handler: handlerFunc(nop)},
	0x01: &instruction{name: "LD BC nn", ticks: 12, length: 3, handler: handlerFunc(ldBcNn)},
	0x02: &instruction{name: "LD (BC) A", ticks: 8, length: 1, handler: handlerFunc(ldBcpA)},
	0x03: &instruction{name: "INC BC", ticks: 8, length: 1, handler: handlerFunc(incBc)},
	0x04: &instruction{name: "INC B", ticks: 4, length: 1, handler: handlerFunc(incB)},
	0x05: &instruction{name: "DEC B", ticks: 4, length: 1, handler: handlerFunc(decB)},
	0x06: &instruction{name: "LD B n", ticks: 8, length: 2, handler: handlerFunc(ldBn)},
	0x07: &instruction{name: "RLCA", ticks: 4, length: 1, handler: handlerFunc(rlca)},
	0x08: &instruction{name: "LD (nn) SP", ticks: 20, length: 3, handler: handlerFunc(ldNnpSp)},
	0x09: &instruction{name: "ADD HL BC", ticks: 8, length: 1, handler: handlerFunc(addHlBc)},
	0x0A: &instruction{name: "LD A, (BC)", ticks: 8, length: 1, handler: handlerFunc(ldABcp)},
	0x0B: &instruction{name: "DEC BC", ticks: 8, length: 1, handler: handlerFunc(decBc)},
	0x0C: &instruction{name: "INC C", ticks: 4, length: 1, handler: handlerFunc(incC)},
	0x0D: &instruction{name: "DEC C", ticks: 4, length: 1, handler: handlerFunc(decC)},
	0x0E: &instruction{name: "LD C n", ticks: 8, length: 2, handler: handlerFunc(ldCN)},
	0x0F: &instruction{name: "RRCA", ticks: 4, length: 1, handler: handlerFunc(rrca)},
}

type cpu struct {
	registers      registers
	ticks          uint64
	instructionSet *map[int]*instruction

	// Processing units
	mmu *mmu
	gpu *gpu
	spu *spu
}

func (cpu *cpu) initialize() {
	cpu.initializeInstructionset()
	cpu.mmu = &mmu{}
	cpu.gpu = &gpu{}
	cpu.spu = &spu{}
}

func (cpu *cpu) initializeInstructionset() error {
	cpu.instructionSet = &instructionSetDeclaration
	for _, instruction := range *cpu.instructionSet {
		instruction.cpu = cpu
	}
	return nil
}

func (cpu *cpu) next() error {
	opcode := cpu.mmu.readByte(cpu.registers.pc)
	cpu.registers.pc++
	inst := (*cpu.instructionSet)[int(opcode)]
	inst.execute()
	return nil
}

func (cpu *cpu) aluRotateLeftCarry(value byte) byte {
	carry := (value & 0x80) >> 7
	value <<= 1
	if (cpu.registers.F & carryFlag) == carryFlag {
		value++
	}

	if carry == 1 {
		cpu.registers.F |= carryFlag
	} else {
		cpu.registers.F ^= carryFlag
	}

	cpu.registers.F ^= negativeFlag | zeroFlag | halfCarryFlag

	return value
}

func (cpu *cpu) aluInc(value byte) byte {
	if value&0x0F == 0 {
		cpu.registers.F |= halfCarryFlag
	} else {
		cpu.registers.F ^= halfCarryFlag
	}

	value++

	if value == 0 {
		cpu.registers.F |= zeroFlag
	} else {
		cpu.registers.F ^= zeroFlag
	}

	cpu.registers.F ^= negativeFlag

	return value
}

func (cpu *cpu) aluDec(value byte) byte {
	if value&0x0F == 0x0 {
		cpu.registers.F ^= halfCarryFlag
	} else {
		cpu.registers.F |= halfCarryFlag
	}

	value--

	if value == 0 {
		cpu.registers.F |= zeroFlag
	} else {
		cpu.registers.F ^= zeroFlag
	}

	cpu.registers.F |= negativeFlag

	return value
}

func (cpu *cpu) addWord(a uint16, b uint16) uint16 {
	if uint(a)+uint(b) > 0xFFFF {
		cpu.registers.F |= carryFlag
	}

	if uint16(a&0x0FFF)+uint16(b&0x0FFF) > 0x0FFF {
		cpu.registers.F |= halfCarryFlag
	}

	cpu.registers.F ^= negativeFlag

	return a + b
}

// Instructions implementation :

func nop(cpu *cpu, _ uint16) {
}

func ldBcNn(cpu *cpu, parameter uint16) {
	cpu.registers.writeBC(parameter)
}

func ldBcpA(cpu *cpu, _ uint16) {
	cpu.mmu.writeByte(cpu.registers.readBC(), cpu.registers.A)
}

func incBc(cpu *cpu, _ uint16) {
	bc := cpu.registers.readBC()
	bc++
	cpu.registers.writeBC(bc)
}

func incB(cpu *cpu, _ uint16) {
	cpu.registers.B = cpu.aluInc(cpu.registers.B)
}

func decB(cpu *cpu, _ uint16) {
	cpu.registers.B = cpu.aluDec(cpu.registers.B)
}

func ldBn(cpu *cpu, parameter uint16) {
	cpu.registers.B = uint8(parameter & 0x00FF)
}

func rlca(cpu *cpu, _ uint16) {
	cpu.registers.A = cpu.aluRotateLeftCarry(cpu.registers.A)
}

func ldNnpSp(cpu *cpu, value uint16) {
	cpu.mmu.writeWord(value, cpu.registers.sp)
}

func addHlBc(cpu *cpu, _ uint16) {
	hl := cpu.registers.readHL()
	bc := cpu.registers.readBC()
	cpu.registers.writeHL(cpu.addWord(hl, bc))
}

func ldABcp(cpu *cpu, address uint16) {
	cpu.registers.A = cpu.mmu.readByte(cpu.registers.readBC())
}

func decBc(cpu *cpu, _ uint16) {
	value := cpu.registers.readBC()
	value--
	cpu.registers.writeBC(value)
}

func incC(cpu *cpu, _ uint16) {
	cpu.registers.C = cpu.aluInc(cpu.registers.C)
}

func decC(cpu *cpu, _ uint16) {
	cpu.registers.C = cpu.aluDec(cpu.registers.C)
}

func ldCN(cpu *cpu, value uint16) {
	cpu.registers.C = byte(value & 0x00FF)
}

func rrca(cpu *cpu, _ uint16) {
	if cpu.registers.A&0x01 == 0x01 {
		cpu.registers.F |= carryFlag
	}
	cpu.registers.A = cpu.registers.A >> 1
}
