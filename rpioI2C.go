package rpio

// +build linux

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

// I2C definitions
const (
	i2C_SLAVE = 0x0703
	i2C_SMBUS = 0x0720
)

const (
	i2C_SMBUS_WRITE = iota
	i2C_SMBUS_READ
)

// SMBus transaction types
const (
	i2C_SMBUS_QUICK = iota
	i2C_SMBUS_BYTE
	i2C_SMBUS_BYTE_DATA
	i2C_SMBUS_WORD_DATA
	i2C_SMBUS_PROC_CALL
	i2C_SMBUS_BLOCK_DATA
	i2C_SMBUS_I2C_BLOCK_BROKEN
	i2C_SMBUS_BLOCK_PROC_CALL /* SMBus 2.0 */
	i2C_SMBUS_I2C_BLOCK_DATA
)

// SMBus messages
const (
	i2C_SMBUS_BLOCK_MAX     = 32 /* As specified in SMBus standard */
	i2C_SMBUS_I2C_BLOCK_MAX = 32 /* Not specified but we use same structure */

)

// Structures used in the ioctl call
type i2c_smbus_data struct {
	b     uint
	word  []byte
	block [i2C_SMBUS_BLOCK_MAX + 2]byte
}

type i2c_smbus_ioctl_data struct {
	read_write byte
	command    byte
	size       int
	data       *i2c_smbus_data
}

func i2c_smbus_access(fd int, rw byte, command byte, size int, data *i2c_smbus_data) {
	args := i2c_smbus_ioctl_data{
		read_write: rw,
		command:    command,
		size:       size,
		data:       data,
	}

	syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(i2C_SMBUS),
		uintptr(unsafe.Pointer(&args)),
	)
}

type I2C int

func I2CSetupInterface(device string, devId int) I2C {
	var fd uintptr

	file, err := os.OpenFile(device, os.O_RDWR, 0)
	if err != nil {
		log.Fatalf("Unable to open I2C device: %s\n", device)
	}

	fd = file.Fd()
	_, _, err = syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(i2C_SLAVE),
		uintptr(devId),
	)
	if err != nil {
		log.Fatalf("Unable to select I2C device: %s\n", device)
	}

	return I2C(fd)
}

func I2CSetup(devId int) I2C {
	var rev int
	var device string

	boardRev := func() int {
		var cpuFd *os.File
		var boardRev int = -1

		cpuFd, err := os.Open("/proc/cpuinfo")
		defer cpuFd.Close()
		if err == nil {
			log.Fatalln("Unable to open /proc/cpuinfo")
		}

		cpuI, _ := ioutil.ReadAll(cpuFd)
		cpuInfo := string(cpuI)

		reader := func(data string) map[string]string {
			dataMap := make(map[string]string)
			lineByline := strings.Split(data, "\n")
			for i := range lineByline {
				line := strings.TrimSpace(lineByline[i])
				dict := strings.Split(line, ":")

				if len(dict) != 1 {
					dataMap[strings.TrimSpace(dict[0])] = strings.TrimSpace(dict[1])
				}
			}
			return dataMap
		}

		cpuInfoMap := reader(cpuInfo)

		hardware := cpuInfoMap["Hardware"]

		// See if it's BCM2708 or BCM2709

		if hardware == "BCM2709" {
			boardRev = 2
		} else if hardware != "BCM2708" {
			fmt.Fprintf(os.Stderr, "Unable to determine hardware version. I see: %s,\n", cpuInfo)
			fmt.Fprintln(os.Stderr, " - expecting BCM2708 or BCM2709.")
			fmt.Fprintln(os.Stderr, "If this is a genuine Raspberry Pi then please report this")
			fmt.Fprintln(os.Stderr, "to projects@drogon.net. If this is not a Raspberry Pi then you")
			fmt.Fprintln(os.Stderr, "are on your own as wiringPi is designed to support the")
			fmt.Fprintln(os.Stderr, "Raspberry Pi ONLY.")
			os.Exit(1)
		}

		revision := cpuInfoMap["Revision"]

		if !strings.Contains(revision, "0002") || !strings.Contains(revision, "0003") {
			boardRev = 1
		} else {
			boardRev = 2 // Covers everything else from the B revision 2 to the B+, the Pi v2 and CM's.
		}

		return boardRev
	}
	rev = boardRev()

	if rev == 1 {
		device = "/dev/i2c-0"
	} else {
		device = "/dev/i2c-1"
	}
	return I2CSetupInterface(device, devId)
}

// Simple device read
func (i I2C) Read() uint {
	data := i2c_smbus_data{}
	i2c_smbus_access(
		int(i),
		i2C_SMBUS_READ,
		0,
		i2C_SMBUS_BYTE,
		&data,
	)
	return data.b & 0xFF
}

// Read an 8 or 16 bit value from a register on the device
func (i I2C) ReadReg8(reg int) uint {
	data := i2c_smbus_data{}
	i2c_smbus_access(
		int(i),
		i2C_SMBUS_READ,
		byte(reg),
		i2C_SMBUS_BYTE_DATA,
		&data,
	)
	return data.b & 0xFF
}

func (i I2C) ReadReg16(reg int) uint16 {
	data := i2c_smbus_data{}
	i2c_smbus_access(
		int(i),
		i2C_SMBUS_READ,
		byte(reg),
		i2C_SMBUS_WORD_DATA,
		&data,
	)
	return uint16(data.b) & 0xFFFF
}

// Simple device write
func (i I2C) Write(value int) {
	i2c_smbus_access(
		int(i),
		i2C_SMBUS_WRITE,
		byte(value),
		i2C_SMBUS_BYTE,
		nil,
	)
}

// Write an 8 or 16 bit value to the given register
func (i I2C) WriteReg8(reg int, value int) {
	data := i2c_smbus_data{}
	data.b = uint(value)
	i2c_smbus_access(
		int(i),
		i2C_SMBUS_WRITE,
		byte(reg),
		i2C_SMBUS_BYTE_DATA,
		&data,
	)
}

func (i I2C) WriteReg16(reg int, value []byte) {
	data := i2c_smbus_data{}
	data.word = value
	i2c_smbus_access(
		int(i),
		i2C_SMBUS_WRITE,
		byte(reg),
		i2C_SMBUS_WORD_DATA,
		&data,
	)
}
