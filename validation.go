package main

import (
	"encoding/json"
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

// OvpnFileOutput is the openvpn file output struct
type OvpnFileOutput struct {
	ok    bool
	ofile OvpnFile
}

// CmdOutput Command output struct
type CmdOutput struct {
	data string
	err  error
}

const configFile = "config.json"
const op1 = "--config"
const op2 = "--auth-user-pass"

var bin string
var sfolder string
var tfolder string
var pwd string
var timeout time.Duration
var thread int

func main() {

	bin, sfolder, tfolder, pwd, timeout, thread = parseJSON()

	cleanFolder()

	fl := getFiles(sfolder)
	size := len(fl)

	ic := make(chan OvpnFile, size)
	oc := make(chan OvpnFileOutput, size)

	for _, f := range fl {
		ic <- f
	}

	close(ic)

	for w := 0; w < thread; w++ {
		go process(ic, oc)
	}

	for i := 0; i < cap(oc); i++ {
		o := <-oc
		if o.ok {
			o.ofile.copy()
		}
	}

	fmt.Println("Done")
}

func cleanFolder() {
	os.RemoveAll(tfolder)
	os.Mkdir(tfolder, 0700)
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
	if pwd == "" {
		cmd = exec.Command(bin, op1, of.path, op2, pwd)
	} else {
		cmd = exec.Command(bin, of.path)
	}
	return
}

func (of OvpnFile) copy() {
	d, _ := ioutil.ReadFile(of.path)
	ioutil.WriteFile(filepath.Join(tfolder, of.name), d, 0700)
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
	return
}

func process(inputc <-chan OvpnFile, outputc chan<- OvpnFileOutput) {
	for i := range inputc {
		c := i.composeCmd()
		ok := runCmdTimeout(c, timeout)
		outputc <- OvpnFileOutput{ok, i}
	}
}

func parseJSON() (bin string, sfolder string, tfolder string, pwd string, timeout time.Duration, thread int) {
	m := make(map[string]interface{})
	path, _ := filepath.Abs(configFile)
	b, _ := ioutil.ReadFile(path)
	json.Unmarshal(b, &m)
	bin = m["openvpn"].(string)
	sfolder = m["source folder"].(string)
	tfolder = m["target folder"].(string)
	pwd = m["password file"].(string)
	timeout = time.Duration(m["timeout"].(float64)) * time.Second
	thread = int(m["thread"].(float64))
	return
}

// {
//     "openvpn":"/usr/local/Cellar/openvpn/2.4.6/sbin/openvpn",
//     "source folder":"/Users/qifan/Desktop/ProtonVPN config",
//     "target folder":"/Users/qifan/Desktop/ProtonVPN config/validated",
//     "password file":"/Users/qifan/pass",
//     "tiemout":10,
//     "thread":10
// }
