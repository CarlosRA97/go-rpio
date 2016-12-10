package rpio

// +build linux

import (
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/CarlosRA97/go-rpio/ioctl"
)

// ioctl constants
const (
	TCGETS  = 0x5401
	TCSETS  = 0x5402
	TCSANOW = 0x0
)

type Serial int

func OpenSerial(device string, baud int) Serial {
	var myBaud uint64
	var status int
	var fd int

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

	file, err := os.OpenFile(device, syscall.O_RDWR|syscall.O_NOCTTY|syscall.O_NDELAY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return -1
	}
	fd = int(file.Fd())
	syscall.Syscall(
		syscall.SYS_FCNTL,
		uintptr(fd),
		syscall.F_SETFL,
		0,
	)

	// like tcgetattr() C function
	ioctl.IOCTL(
		uintptr(fd),
		uintptr(syscall.TIOCGETA),
		uintptr(unsafe.Pointer(&options)),
	)

	options.Ispeed = myBaud
	options.Ospeed = myBaud

	options.Cflag |= (syscall.CLOCAL | syscall.CREAD)
	options.Cflag &= 0xFFF
	options.Cflag &= 0x3FF
	options.Cflag &= 0xFF
	options.Cflag |= syscall.CS8
	options.Lflag &= 0x75 //~(syscall.ICANON | syscall.ECHO | syscall.ECHOE | syscall.ISIG)
	options.Oflag &= 0x0

	options.Cc[syscall.VMIN] = 0
	options.Cc[syscall.VTIME] = 100

	// like tcsetattr() C function
	ioctl.IOCTL(
		uintptr(fd),
		uintptr(syscall.TIOCSETAF), //TCSANOW|syscall.TCSAFLUSH
		uintptr(unsafe.Pointer(&options)),
	)

	ioctl.IOCTL(
		uintptr(fd),
		uintptr(TCGETS),
		uintptr(unsafe.Pointer(&status)),
	)

	status |= syscall.TIOCM_DTR
	status |= syscall.TIOCM_RTS

	ioctl.IOCTL(
		uintptr(fd),
		uintptr(TCSETS),
		uintptr(unsafe.Pointer(&status)),
	)

	time.Sleep(10 * time.Millisecond)

	return Serial(fd)
}

func (s Serial) Flush() {

	const FREAD byte = 0x01
	const FWRITE byte = 0x02

	ioctl.IOCTL(
		uintptr(s),
		uintptr(syscall.TCIOFLUSH),
		uintptr(FREAD|FWRITE),
	)
}

func (s Serial) Close() {
	syscall.Close(int(s))
}

func (s Serial) Puts(message string) {
	syscall.Write(int(s), []byte(message))
}

func (s Serial) DataAvail() int {
	var result int
	err := ioctl.IOCTL(
		uintptr(s),
		uintptr(0),
		uintptr(unsafe.Pointer(&result)),
	)

	if err != nil {
		log.Fatalln("Cant read data available")
	}

	return result
}

func (s Serial) GetChar() []byte {
	var x []byte
	_, err := syscall.Read(int(s), x)
	if err != nil {
		return x
	}
	return x
}
