package rpio

import (
	"os"
	"syscall"
	"unsafe"
	"time"
)

// ioctl constants
const (
	TCGETS = 0x5401
	TCSETS = 0x5402
	TCSANOW = 0x0
)

type Serial int

func OpenSerial(device string, baud int) (fd Serial) {
	var myBaud byte
	var status, fd int

	options := syscall.Termios{}

	switch baud {
	case 50:
		myBaud = 0x32
	case 75:
		myBaud = 0x4B
	case 110:
		myBaud = 0x6E
	case 134:
		myBaud = 0x86
	case 150:
		myBaud = 0x96
	case 200:
		myBaud = 0xC8
	case 300:
		myBaud = 0x12C
	case 600:
		myBaud = 0x258
	case 1200:
		myBaud = 0x4B0
	case 1800:
		myBaud = 0x708
	case 2400:
		myBaud = 0x960
	case 4800:
		myBaud = 0x12C0
	case 9600:
		myBaud = 0x2580
	case 19200:
		myBaud = 0x4B00
	case 38400:
		myBaud = 0x9600
	case 57600:
		myBaud = 0xE100
	case 115200:
		myBaud = 0x1C200
	case 230400:
		myBaud = 0x38400

	default:
		return -2
	}

	file, err := os.OpenFile(device, os.O_RDWR|syscall.O_NOCTTY|syscall.O_NDELAY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return
	}
	fd = file.Fd()
	syscall.FcntlFlock(fd, syscall.F_SETFL, syscall.O_RDWR)

	syscall.Syscall(
		syscall.SYS_GETATTRLIST,
		uintptr(fd),
		uintptr(TCSANOW),
		uintptr(unsafe.Pointer(options)),
	)

	options.Ispeed = myBaud
	options.Ospeed = myBaud

	options.Cflag |= (syscall.CLOCAL | syscall.CREAD)
	options.Cflag &= !syscall.PARENB
	options.Cflag &= !syscall.CSTOPB
	options.Cflag &= !syscall.CSIZE
	options.Cflag |= syscall.CS8
	options.Lflag &= !(syscall.ICANON | syscall.ECHO | syscall.ECHOE | syscall.ISIG)
	options.Oflag &= !syscall.OPOST

	options.Cc[syscall.VMIN] = 0
	options.Cc[syscall.VTIME] = 100

	syscall.Syscall(
		syscall.SYS_SETATTRLIST,
		uintptr(fd),
		uintptr(TCSANOW|syscall.TCSAFLUSH),
		uintptr(unsafe.Pointer(options)),
	)

	syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(TCGETS),
		uintptr(unsafe.Pointer(status)),
	)

	status |= syscall.TIOCM_DTR
	status |= syscall.TIOCM_RTS

	syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(TCSETS),
		uintptr(unsafe.Pointer(status)),
	)

	time.Sleep(10*time.Millisecond)

	return fd
}

func (s Serial) Flush() {
	syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(s),
		uintptr(syscall.TCIOFLUSH),
		uintptr(0),
	)
}

func (s Serial) Close() {
	syscall.Close(s)
}

func (s Serial) Puts(message string) {
	syscall.Write(s, []byte(message))
}

func (s Serial) DataAvail() (result int) {
	var result int

	_, _ , err := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(s),
		uintptr(0),
		uintptr(unsafe.Pointer(result)),
	)

	if err == -1 {
		return -1
	}

	return result
}

func (s Serial) GetChar() (x []byte) {
	var x []byte
	_, err := syscall.Read(s, &x)
	if err != nil {
		return -1
	}
	return x
}