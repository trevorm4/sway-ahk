package main

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

const (
	uinputPath = "/dev/uinput"

	uiSetEvBit  = 0x40045564 // _IOW('U', 100, int)
	uiSetKeyBit = 0x40045565 // _IOW('U', 101, int)
	uiDevCreate = 0x5501     // _IO('U', 1)

	evSyn     = 0x00
	evKey     = 0x01
	synReport = 0x00
)

type inputEvent struct {
	Sec   int64
	Usec  int64
	Type  uint16
	Code  uint16
	Value int32
}

type uinputID struct {
	BusType uint16
	Vendor  uint16
	Product uint16
	Version uint16
}

type uinputUserDev struct {
	Name         [80]byte
	ID           uinputID
	FFEffectsMax int32
	AbsMax       [64]int32
	AbsMin       [64]int32
	AbsFuzz      [64]int32
	AbsFlat      [64]int32
}

type uinputKeyboard struct {
	mu sync.Mutex
	fd int
}

func openUinputKeyboard(name string) (*uinputKeyboard, error) {
	fd, err := syscall.Open(uinputPath, syscall.O_WRONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", uinputPath, err)
	}

	kb := &uinputKeyboard{fd: fd}

	fail := func(step string, err error) (*uinputKeyboard, error) {
		syscall.Close(fd)
		return nil, fmt.Errorf("%s: %w", step, err)
	}

	if err := ioctlInt(fd, uiSetEvBit, evSyn); err != nil {
		return fail("set ev_syn bit", err)
	}
	if err := ioctlInt(fd, uiSetEvBit, evKey); err != nil {
		return fail("set ev_key bit", err)
	}
	for _, code := range keyCodeMap {
		if err := ioctlInt(fd, uiSetKeyBit, int32(code)); err != nil {
			return fail(fmt.Sprintf("set key bit %d", code), err)
		}
	}

	dev := uinputUserDev{ID: uinputID{BusType: 0x03, Vendor: 0x1234, Product: 0x5678, Version: 1}}
	copy(dev.Name[:], name)
	if _, err := syscall.Write(fd, (*[unsafe.Sizeof(dev)]byte)(unsafe.Pointer(&dev))[:]); err != nil {
		return fail("write device info", err)
	}

	if err := ioctl(fd, uiDevCreate, 0); err != nil {
		return fail("create device", err)
	}
	return kb, nil
}

func (k *uinputKeyboard) TapKey(code int) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	if err := k.emit(evKey, uint16(code), 1); err != nil {
		return err
	}
	if err := k.emit(evSyn, synReport, 0); err != nil {
		return err
	}
	if err := k.emit(evKey, uint16(code), 0); err != nil {
		return err
	}
	return k.emit(evSyn, synReport, 0)
}

func (k *uinputKeyboard) Close() error {
	k.mu.Lock()
	defer k.mu.Unlock()
	return syscall.Close(k.fd)
}

func (k *uinputKeyboard) emit(evType, code uint16, value int32) error {
	e := inputEvent{Type: evType, Code: code, Value: value}
	_, err := syscall.Write(k.fd, (*[unsafe.Sizeof(e)]byte)(unsafe.Pointer(&e))[:])
	return err
}

func ioctl(fd int, req uintptr, arg uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), req, arg)
	if errno != 0 {
		return errno
	}
	return nil
}

func ioctlInt(fd int, req uintptr, value int32) error {
	return ioctl(fd, req, uintptr(value))
}
