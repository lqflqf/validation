package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"github.com/cheggaaa/pb"
)

// OvpnFile is the openvpn file struct
type OvpnFile struct {
	path    string
	name    string
	modtime time.Time
	extra ExtracInfo
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

//ExtracInfo struct
type ExtracInfo struct {
	country string
	score   int
	connectInfo string
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
	fl := getFiles(sfolder)
	fl = removeDup(fl)
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

	// find out working files
	ofl := make([]OvpnFile, 0)

	for i := 0; i < cap(oc); i++ {
		o := <-oc
		if o.ok {
			ofl = append(ofl, o.ofile)
		}
	}

	//sort by score
	sort.Slice(ofl, func(i, j int) bool {
		return ofl[i].extra.score > ofl[j].extra.score
	})

	//renanem and copy
	digitFmt := "%0" + strconv.Itoa(getDigitNumber(len(ofl))) + "d"
	var t int
	for i, f := range ofl {
		strRank := fmt.Sprintf(digitFmt, i+1)
		f.name = strRank + "_" + f.extra.country + ".ovpn"
		e := f.copy()
		if e == nil {
			t++
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
				filestat, _ := os.Stat(path)
				modtime := filestat.ModTime()
				fl = append(fl, OvpnFile{path, info.Name(), modtime, getExtrainfo(info.Name())})
			}
		}
		return nil
	}
	filepath.Walk(folder, wf)
	return
}

func removeDup(dl []OvpnFile) (ndl []OvpnFile) {
	l := len(dl)
	m := make(map[string]OvpnFile)
	for i := 0; i < l; i++ {
		ovpnfile, ok := m[dl[i].extra.connectInfo]
		if ok == false {
			m[dl[i].extra.connectInfo] = dl[i]
		} else {
			if ovpnfile.modtime.Before(dl[i].modtime) {
				m[dl[i].extra.connectInfo] = dl[i]
			}
		}
	}
	for _, v := range m {
		ndl = append(ndl, v)
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

func (of OvpnFile) copy() error {
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
	if lvalidstr, ok := m["valid string"].(string); ok {
		validstr = lvalidstr
	} else {
		validstr = "Cannot allocate TUN/TAP dev dynamically"
	}
	pwd = m["password file"].(string)
	timeout = time.Duration(m["timeout"].(float64))
	thread = int(m["thread"].(float64))
}

func getExtrainfo(fileName string) ExtracInfo {
	absName := strings.Split(fileName, ".ovpn")[0]
	nl := strings.Split(absName, "_")
	strScore := nl[4]
	score, _ := strconv.Atoi(strScore)
	return ExtracInfo{nl[0], score, nl[1] + "_"+ nl[2] + "_" + nl[3]}
}

func getDigitNumber(fileNo int) int {
	strFileNo := strconv.Itoa(fileNo)
	return len(strFileNo)
}
