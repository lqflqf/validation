package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type file struct {
	path string
	name string
}

type output struct {
	data string
	err  error
}

const op1 = "--config"
const op2 = "--auth-user-pass"

var bin string
var sfolder string
var tfolder string
var pwd string
var pflag bool
var timeout time.Duration
var thread int

func main() {

	// ovpnBin := "/usr/local/Cellar/openvpn/2.4.6/sbin/openvpn"
	// ovpnFile := "/Users/qifan/Desktop/ProtonVPN config/ProtonVPN_server_configs_UDP/us-va-110.protonvpn.com.udp1194.ovpn"
	// passFile := "/Users/qifan/pass"
	// ovpnCmd := exec.Command(ovpnBin, op1, ovpnFile, op2, passFile)

	// ok := runCmdTimeout(ovpnCmd, 10)
	// fmt.Println(ok)

	bin = "/usr/local/Cellar/openvpn/2.4.6/sbin/openvpn"
	sfolder = "/Users/qifan/Desktop/ProtonVPN config/ProtonVPN_server_configs_UDP"
	pwd = "/Users/qifan/pass"
	pflag = true
	timeout = time.Duration(15)
	thread = 20

	fl := getFiles(sfolder)
	size := len(fl)
	ic := make(chan *exec.Cmd, size)
	oc := make(chan bool, size)

	for _, f := range fl {
		ic <- composeCmd(f.path)
	}

	close(ic)

	for w := 0; w < thread; w++ {
		go process(ic, oc)
	}

	for i := 0; i < cap(oc); i++ {
		fmt.Println(<-oc)
	}
}

func runCmdTimeout(cmd *exec.Cmd, timeout time.Duration) (ok bool) {
	cOutput := make(chan output)
	go func() {
		o, e := cmd.Output()
		cOutput <- output{string(o), e}
	}()

	select {
	case o := <-cOutput:
		sl := strings.Split(o.data, "\n")
		ok = strings.Contains(sl[cap(sl)-3], "Cannot allocate TUN/TAP dev dynamically")
	case <-time.After(timeout * time.Second):
		cmd.Process.Kill()
		ok = false
	}
	return
}

func getFiles(folder string) (fl []file) {
	var wf filepath.WalkFunc
	wf = func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			n := strings.Split(info.Name(), ".")
			if n[cap(n)-1] == "ovpn" {
				fl = append(fl, file{path, info.Name()})
			}
		}
		return nil
	}
	filepath.Walk(folder, wf)
	return
}

func composeCmd(file string) (cmd *exec.Cmd) {
	if pflag {
		cmd = exec.Command(bin, op1, file, op2, pwd)
	} else {
		cmd = exec.Command(bin, file)
	}
	return
}

func process(inputc <-chan *exec.Cmd, outputc chan<- bool) {
	for i := range inputc {
		outputc <- runCmdTimeout(i, timeout)
	}
}
