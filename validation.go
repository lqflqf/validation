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

	pb "gopkg.in/cheggaaa/pb.v1"
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
var validstr string
var pwd string
var timeout time.Duration
var thread int

func main() {

	arg := os.Args

	if len(arg) == 1 {
		parseJSON(configFile)
	} else {
		parseJSON(arg[1])
	}

	cleanFolder()

	fl := removeDup(getFiles(sfolder))
	size := len(fl)

	ic := make(chan OvpnFile, size)
	oc := make(chan OvpnFileOutput, size)

	for _, f := range fl {
		ic <- f
	}

	close(ic)

	var bar = pb.New(size)

	fmt.Println("In progress...")
	fmt.Println()

	for w := 0; w < thread; w++ {
		go process(ic, oc, bar)
	}

	var t int
	for i := 0; i < cap(oc); i++ {
		o := <-oc
		if o.ok {		
			e := o.ofile.copy()
			if e == nil {
				t++
			}
		}
	}

	bar.FinishPrint("Validation Done!")
	fmt.Printf("%d files are valid", t)
	fmt.Println()
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

func removeDup(dl []OvpnFile) (ndl []OvpnFile) {
	l := len(dl)
	m := make(map[string]bool)
	for i := 0; i < l; i++ {
		_, ok := m[dl[i].name]
		if ok == false {
			m[dl[i].name] = true
			ndl = append(ndl, dl[i])
		}
	}
	return
}

func (of OvpnFile) composeCmd() (cmd *exec.Cmd) {
	if pwd != "" {
		cmd = exec.Command(bin, op1, of.path, op2, pwd)
	} else {
		cmd = exec.Command(bin, of.path)
	}
	return
}

func (of OvpnFile) copy() error{
	d, _ := ioutil.ReadFile(of.path)
	return ioutil.WriteFile(filepath.Join(tfolder, of.name), d, 0700)
}

func runCmdTimeout(cmd *exec.Cmd) (ok bool) {
	cOutput := make(chan CmdOutput)
	go func() {
		o, e := cmd.Output()
		cOutput <- CmdOutput{string(o), e}
	}()

	select {
	case o := <-cOutput:
		sl := strings.Split(o.data, "\n")
		ok = strings.Contains(sl[cap(sl)-3], validstr)
	case <-time.After(timeout * time.Second):
		cmd.Process.Kill()
		ok = false
	}
	return
}

func process(inputc <-chan OvpnFile, outputc chan<- OvpnFileOutput, pb *pb.ProgressBar) {
	pb.Start()
	for i := range inputc {
		c := i.composeCmd()
		ok := runCmdTimeout(c)
		outputc <- OvpnFileOutput{ok, i}
		pb.Increment()
	}
}

func parseJSON(cfilename string) {
	m := make(map[string]interface{})
	path, _ := filepath.Abs(cfilename)
	b, _ := ioutil.ReadFile(path)
	json.Unmarshal(b, &m)
	bin = m["openvpn"].(string)
	sfolder = m["source folder"].(string)
	tfolder = m["target folder"].(string)
	validstr = m["valid string"].(string)
	pwd = m["password file"].(string)
	timeout = time.Duration(m["timeout"].(float64))
	thread = int(m["thread"].(float64))
	return
}
