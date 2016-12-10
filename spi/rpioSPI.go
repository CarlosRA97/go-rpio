package spi

import (
	"syscall"
	"unsafe"
)

const (
	spiDev0  string = "/dev/spidev0.0"
	spiDev1  string = "/dev/spidev0.1"
	spiBPW   uint8  = 8
	spiDelay uint16 = 0
)

var (
	spiSpeeds [2]uint32
	spiFds    [2]int
)

type spiIocTransfer struct {
	tx_buf uint64
	rx_buf uint64

	len      uint32
	speed_hz uint32

	deley_usecs   uint16
	bits_per_word uint8
	cs_change     uint8
	pad           uint32
}

func SPIGetFd(channel int) int {
	return spiFds[channel&1]
}

func SPIDataRW(channel int, data uint64, _len uint32) int {

	channel &= 1

	spi := spiIocTransfer{
		tx_buf:        data,
		rx_buf:        data,
		len:           _len,
		deley_usecs:   spiDelay,
		speed_hz:      spiSpeeds[channel],
		bits_per_word: spiBPW,
	}

	r1, _, _ := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(spiFds[channel]),
		uintptr(1),
		uintptr(unsafe.Pointer(&spi)),
	)
	return int(r1)
}
