package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// OvpnFile is the openvpn file struct
type OvpnFile struct {
	path string
	name string
}

// CmdOutput Command output struct
type CmdOutput struct {
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
	sfolder = "/Users/qifan/vpngate_config/20181013 154623"
	tfolder = "/Users/qifan/vpngate_config/20181013 154623/validated"
	pwd = "/Users/qifan/pass"
	pflag = false
	timeout = time.Duration(15)
	thread = 10

	cleanFolder()

	fl := getFiles(sfolder)
	size := len(fl)

	ic := make(chan *OvpnFile, size)
	oc := make(chan *OvpnFile, size)

	for _, f := range fl {
		fmt.Println(f.name)
		ic <- &f
	}

	close(ic)

	for w := 0; w < thread; w++ {
		go process(ic, oc)
	}

	for i := 0; i < cap(oc); i++ {
		o := <-oc
		fmt.Println(o.name)
		o.copy()
	}

	fmt.Println("Done")
}

func cleanFolder() {
	os.RemoveAll(tfolder)
	os.Mkdir(tfolder, os.ModeDir)
}

func getFiles(folder string) (fl []OvpnFile) {
	var wf filepath.WalkFunc
	wf = func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			n := strings.Split(info.Name(), ".")
			if n[cap(n)-1] == "ovpn" {
				fl = append(fl, OvpnFile{path, info.Name()})
			}
		}
		return nil
	}
	filepath.Walk(folder, wf)
	return
}

func (of OvpnFile) composeCmd() (cmd *exec.Cmd) {
	fmt.Println(of.path)
	if pflag {
		cmd = exec.Command(bin, op1, of.path, op2, pwd)
	} else {
		cmd = exec.Command(bin, of.path)
	}
	return
}

func (of OvpnFile) copy() {
	d, _ := ioutil.ReadFile(of.path)
	ioutil.WriteFile(filepath.Join(tfolder, of.name), d, 0644)
}

func runCmdTimeout(cmd *exec.Cmd, timeout time.Duration) (ok bool) {
	cOutput := make(chan CmdOutput)
	go func() {
		o, e := cmd.Output()
		cOutput <- CmdOutput{string(o), e}
	}()

	select {
	case o := <-cOutput:
		sl := strings.Split(o.data, "\n")
		ok = strings.Contains(sl[cap(sl)-3], "Cannot allocate TUN/TAP dev dynamically")
	case <-time.After(timeout * time.Second):
		cmd.Process.Kill()
		ok = false
	}
	fmt.Println(ok)
	return
}

func process(inputc <-chan *OvpnFile, outputc chan<- *OvpnFile) {
	for i := range inputc {
		//outputc <- runCmdTimeout(i, timeout)
		cmd := i.composeCmd()
		ok := runCmdTimeout(cmd, timeout)
		if ok {
			outputc <- i
		}
	}
}
