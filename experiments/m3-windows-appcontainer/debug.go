//go:build windows

package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"golang.org/x/sys/windows"
	"time"
	"unsafe"
)

// Temporary diagnostic of startup, bounded to the synthetic browser trial.
func debugBrowser(process windows.Handle, done <-chan error) error {
	kernel := windows.NewLazySystemDLL("kernel32.dll")
	wait := kernel.NewProc("WaitForDebugEvent")
	resume := kernel.NewProc("ContinueDebugEvent")
	firstBreakpoint := true
	var base uintptr
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			return err
		default:
		}
		var event [176]byte
		ok, _, _ := wait.Call(uintptr(unsafe.Pointer(&event[0])), 100)
		if ok == 0 {
			continue
		}
		kind := binary.LittleEndian.Uint32(event[:4])
		pid := binary.LittleEndian.Uint32(event[4:8])
		tid := binary.LittleEndian.Uint32(event[8:12])
		status := uintptr(0x00010002)
		switch kind {
		case 1:
			exception := binary.LittleEndian.Uint32(event[16:20])
			if exception == 0x80000003 && firstBreakpoint {
				firstBreakpoint = false
			} else if exception == 0x406d1388 {
				status = 0x80010001
			} else {
				fmt.Printf("Image base: 0x%x RVA: 0x%x\n", base, uintptr(binary.LittleEndian.Uint64(event[32:40]))-base)
				debugStack(kernel, process, tid, base)
				fmt.Printf("Startup debug exception: code=0x%x address=0x%x first=%d\n", exception, binary.LittleEndian.Uint64(event[32:40]), binary.LittleEndian.Uint32(event[168:172]))
				status = 0x80010001
			}
		case 3, 6:
			if kind == 3 {
				base = uintptr(binary.LittleEndian.Uint64(event[40:48]))
			}
			// CREATE_PROCESS / LOAD_DLL carry an open file handle owned by the debugger.
			h := windows.Handle(binary.LittleEndian.Uint64(event[16:24]))
			if h != 0 {
				windows.CloseHandle(h)
			}
		case 8:
			length := int(binary.LittleEndian.Uint16(event[26:28]))
			unicode := binary.LittleEndian.Uint16(event[24:26]) != 0
			if unicode {
				length *= 2
			}
			if length > 8192 {
				length = 8192
			}
			if length > 0 {
				data := make([]byte, length)
				var n uintptr
				if windows.ReadProcessMemory(process, uintptr(binary.LittleEndian.Uint64(event[16:24])), &data[0], uintptr(length), &n) == nil {
					if unicode {
						units := make([]uint16, len(data)/2)
						for i := range units {
							units[i] = binary.LittleEndian.Uint16(data[i*2:])
						}
						fmt.Printf("Startup debug: %s\n", windows.UTF16ToString(units))
					} else {
						fmt.Printf("Startup debug: %s\n", data[:n])
					}
				}
			}
		}
		resume.Call(uintptr(pid), uintptr(tid), status)
	}
	return errors.New("CDP deadline exceeded")
}

func debugStack(kernel *windows.LazyDLL, process windows.Handle, tid uint32, base uintptr) {
	thread, err := windows.OpenThread(8, false, tid)
	if err != nil {
		return
	}
	defer windows.CloseHandle(thread)
	var storage [1248]byte
	addr := (uintptr(unsafe.Pointer(&storage[0])) + 15) &^ uintptr(15)
	ctx := unsafe.Slice((*byte)(unsafe.Pointer(addr)), 1232)
	binary.LittleEndian.PutUint32(ctx[48:52], 0x100001)
	ok, _, _ := kernel.NewProc("GetThreadContext").Call(uintptr(thread), addr)
	if ok == 0 {
		return
	}
	rsp := uintptr(binary.LittleEndian.Uint64(ctx[152:160]))
	var stack [256]byte
	var n uintptr
	if windows.ReadProcessMemory(process, rsp, &stack[0], uintptr(len(stack)), &n) != nil {
		return
	}
	fmt.Print("Startup stack RVAs:")
	for i := 0; i+8 <= int(n); i += 8 {
		v := uintptr(binary.LittleEndian.Uint64(stack[i : i+8]))
		if v >= base && v < base+200*1024*1024 {
			fmt.Printf(" 0x%x", v-base)
		}
	}
	fmt.Println()
}
