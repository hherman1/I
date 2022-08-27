package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"9fans.net/go/acme"
)

func main() {
	if err := start(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var win *acme.Win
var run struct {
	// Synchronizes access to the output buffer and to the details of the current running command.
	sync.Mutex
	// Which run we are on. Existing runs should relinquish control of the buffer and exit if their run ID does not match the current run ID.
	id  int
	cmd *exec.Cmd
}

func start() error {
	cmd := strings.Join(os.Args[1:], " ")
	if cmd == "" {
		return errors.New("no command given")
	}

	var err error
	win, err = acme.New()
	if err != nil {
		return fmt.Errorf("constuct window: %w", err)
	}
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get wd: %w", err)
	}
	win.Name(filepath.Join(pwd, "+I"))
	win.Write("data", []byte(fmt.Sprintf("%% %v\n", cmd)))
	win.Fprintf("tag", "Get Back")

	err = launch()
	if err != nil {
		return fmt.Errorf("initial launch: %w", err)
	}

	for e := range win.EventChan() {
		if e.C2 == 'X' {
			run.Lock()
			bs, err := win.ReadAll("body")
			if err != nil {
				return fmt.Errorf("read body: %w", err)
			}
			find := -1
			for i, b := range bs {
				find = i
				if b == '\n' {
					break
				}
			}
			win.Addr("#%d", find)
			win.Write("data", []byte{' '})
			win.Write("data", e.Text)
			run.Unlock()
			err = launch()
			if err != nil {
				return fmt.Errorf("launching: %w", err)
			}
			continue
		}
		if e.C2 == 'x' && string(e.Text) == "Back" {
			run.Lock()
			bs, err := win.ReadAll("body")
			if err != nil {
				return fmt.Errorf("read body: %w", err)
			}
			find := bytes.Index(bs, []byte{'\n'})
			if find != -1 {
				bs = bs[:find]
			}
			find = bytes.LastIndex(bs, []byte{' '})
			if find != -1 {
				win.Addr("#%d,", find)
				win.Write("data", []byte{'\n'})
			}
			run.Unlock()
			err = launch()
			if err != nil {
				return fmt.Errorf("launching: %w", err)
			}
			continue
		}
		if e.C2 == 'x' && string(e.Text) == "Get" {
			err := launch()
			if err != nil {
				return fmt.Errorf("Get: launch: %w", err)
			}
			continue
		}
		if e.C2 == 'x' && string(e.Text) == "Del" {
			win.Ctl("delete")
		}
		win.WriteEvent(e)
	}
	return nil
}

// Clears all output except the current command and returns the current command. Does not acquire the run lock.
func clear() (string, error) {

	// Clean input
	bs, err := win.ReadAll("body")
	if err != nil {
		return "", fmt.Errorf("read data: %w", err)
	}
	find := -1
	for i, b := range []rune(string(bs)) {
		if b == '\n' {
			find = i
			break
		}
	}
	if find != -1 {
		win.Addr("#%d,", find+1)
		win.Write("data", nil)
	}
	//win.Addr("#0")
	return strings.TrimSpace(string(bs[1:find])), nil
}

// Triggers a new launch of the current command, killing any existing command
func launch() error {
	run.Lock()
	run.id++
	next := run.id
	old := run.cmd
	c, err := clear()
	run.Unlock()
	// Nothing should be writing to the buffer now, as we have advanced the run ID
	if err != nil {
		return fmt.Errorf("clearing: %w", err)
	}
	// kill existing
	if old != nil && old.Process != nil {
		err = old.Process.Kill()
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("killing old process: %w", err)
		}
	}

	go execute(next, c)
	return nil
}

// Actually executes a given command, storing it in the run struct
func execute(id int, c string) {
	cmd := exec.Command("bash", "-c", c)
	r, w, err := os.Pipe()
	if err != nil {
		run.Lock()
		if run.id == id {
			win.Write("data", []byte(fmt.Sprintf("make pipe for run %v: %v\n", id, err)))
		}
		run.Unlock()
		return
	}
	cmd.Stdout = w
	cmd.Stderr = w
	err = cmd.Start()
	w.Close()

	run.Lock()
	if run.id != id {
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		r.Close()
		run.Unlock()
		return
	}
	if err != nil {
		r.Close()
		run.Unlock()
		win.Write("data", []byte(fmt.Sprintf("Error starting command: %v\n", err)))
		return
	}
	run.cmd = cmd
	run.Unlock()

	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if err == io.EOF {
			break
		}
		run.Lock()
		if err != nil {
			if id != run.id {
				run.Unlock()
				break
			}
			win.Write("data", []byte(fmt.Sprintf("(read from exec: %v)\n", err)))
			run.Unlock()
			break
		}
		if id == run.id && n > 0 {
			win.Write("data", buf[:n])
		}
		run.Unlock()
	}
	err = cmd.Wait()
	run.Lock()
	if err != nil {
		if run.id == id {
			win.Write("data", []byte(fmt.Sprintf("wait for command to exit: %v\n", err)))
		}
	}
	if run.id == id {
		win.Ctl("clean")
	}
	run.Unlock()
}
