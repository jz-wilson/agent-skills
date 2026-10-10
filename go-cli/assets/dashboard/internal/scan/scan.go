// Package scan lists listening TCP sockets and the processes that own them by
// reading a proc filesystem. It returns plain data and knows nothing about the TUI.
package scan

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// stateListen is the kernel's TCP_LISTEN state as printed in /proc/net/tcp.
const stateListen = "0A"

// Socket is one listening TCP socket. PID is 0 when the owner could not be inspected.
type Socket struct {
	Proto   string `json:"proto"`
	Addr    string `json:"addr"`
	Port    int    `json:"port"`
	PID     int    `json:"pid,omitempty"`
	Process string `json:"process,omitempty"`
	Cmdline string `json:"cmdline,omitempty"`
	Inode   uint64 `json:"inode"`
}

type owner struct {
	pid     int
	process string
	cmdline string
}

// Listening scans procRoot (normally "/proc") and returns listening TCP sockets,
// IPv4 and IPv6, sorted by port, then address. Sockets whose owner is not
// readable by the current user are still returned, with PID 0.
func Listening(procRoot string) ([]Socket, error) {
	socks, err := readTable(filepath.Join(procRoot, "net", "tcp"), "tcp", false)
	if err != nil {
		return nil, err
	}
	// IPv6 may be disabled, in which case the file does not exist.
	socks6, err := readTable(filepath.Join(procRoot, "net", "tcp6"), "tcp6", true)
	if err != nil {
		return nil, err
	}
	socks = append(socks, socks6...)
	if len(socks) == 0 {
		return socks, nil
	}

	owners, err := inodeOwners(procRoot)
	if err != nil {
		return nil, err
	}
	for i := range socks {
		if o, ok := owners[socks[i].Inode]; ok {
			socks[i].PID, socks[i].Process, socks[i].Cmdline = o.pid, o.process, o.cmdline
		}
	}
	sort.SliceStable(socks, func(i, j int) bool {
		if socks[i].Port != socks[j].Port {
			return socks[i].Port < socks[j].Port
		}
		return socks[i].Addr < socks[j].Addr
	})
	return socks, nil
}

func readTable(path, proto string, optional bool) ([]Socket, error) {
	f, err := os.Open(path)
	if err != nil {
		if optional && errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close() //nolint:errcheck // read-only: a close error loses no data

	var socks []Socket
	sc := bufio.NewScanner(f)
	sc.Scan() // header line
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 10 || fields[3] != stateListen {
			continue
		}
		addr, port, err := parseAddr(fields[1])
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: inode %q: %w", path, fields[9], err)
		}
		socks = append(socks, Socket{Proto: proto, Addr: addr, Port: port, Inode: inode})
	}
	return socks, sc.Err()
}

// parseAddr decodes "0100007F:1F90" (IPv4) or a 32-hex-digit IPv6 form. The
// kernel prints each 32-bit word of the address in host byte order.
func parseAddr(s string) (string, int, error) {
	hexAddr, hexPort, ok := strings.Cut(s, ":")
	if !ok {
		return "", 0, fmt.Errorf("bad address %q", s)
	}
	port, err := strconv.ParseUint(hexPort, 16, 16)
	if err != nil {
		return "", 0, fmt.Errorf("bad port in %q: %w", s, err)
	}
	if len(hexAddr) != 8 && len(hexAddr) != 32 {
		return "", 0, fmt.Errorf("bad address length in %q", s)
	}
	var raw [16]byte
	words := len(hexAddr) / 8
	for i := range words {
		v, err := strconv.ParseUint(hexAddr[i*8:i*8+8], 16, 32)
		if err != nil {
			return "", 0, fmt.Errorf("bad address in %q: %w", s, err)
		}
		binary.NativeEndian.PutUint32(raw[i*4:], uint32(v))
	}
	if words == 1 {
		return netip.AddrFrom4([4]byte(raw[:4])).String(), int(port), nil
	}
	return netip.AddrFrom16(raw).String(), int(port), nil
}

// inodeOwners maps socket inodes to the lowest PID holding them open.
// Processes that vanish or deny access are skipped; any other error is returned.
func inodeOwners(procRoot string) (map[uint64]owner, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, err
	}
	owners := make(map[uint64]owner)
	for _, e := range entries { // ReadDir sorts by name, not numerically; compare PIDs below
		pid, err := strconv.Atoi(e.Name())
		if err != nil || !e.IsDir() {
			continue
		}
		fdDir := filepath.Join(procRoot, e.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		}
		var o *owner
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue // fd closed between ReadDir and Readlink
			}
			inode, ok := socketInode(target)
			if !ok {
				continue
			}
			if o == nil {
				o = &owner{
					pid:     pid,
					process: readTrim(filepath.Join(procRoot, e.Name(), "comm")),
					cmdline: Cmdline(procRoot, pid),
				}
			}
			if prev, taken := owners[inode]; !taken || pid < prev.pid {
				owners[inode] = *o
			}
		}
	}
	return owners, nil
}

// socketInode parses the "socket:[12345]" target of an fd symlink.
func socketInode(target string) (uint64, bool) {
	s, ok := strings.CutPrefix(target, "socket:[")
	if !ok {
		return 0, false
	}
	s, ok = strings.CutSuffix(s, "]")
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseUint(s, 10, 64)
	return n, err == nil
}

// Cmdline returns /proc/<pid>/cmdline with NUL separators replaced by spaces,
// or "" when it is unreadable or empty (kernel threads, other users' processes).
func Cmdline(procRoot string, pid int) string {
	b, err := os.ReadFile(filepath.Join(procRoot, strconv.Itoa(pid), "cmdline"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(strings.ReplaceAll(string(b), "\x00", " "))
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}
