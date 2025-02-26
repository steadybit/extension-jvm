package starttime

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/tklauser/go-sysconf"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"strconv"
	"strings"
	"time"
)

var clockTicks = int64(100)

func init() {
	if t, err := sysconf.Sysconf(sysconf.SC_CLK_TCK); err == nil {
		clockTicks = int64(t)
	}
}

type Time int64

func Now() Time {
	var info unix.Sysinfo_t
	if err := unix.Sysinfo(&info); err != nil {
		panic(err)
	}
	return Time(info.Uptime)
}

func (t Time) After(o Time) bool {
	return t > o
}

func (t Time) Sub(o Time) time.Duration {
	return time.Duration((int64(t) - int64(o)) * int64(time.Nanosecond) / clockTicks)
}

func ForProcess(p *process.Process) (Time, error) {
	stat, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", p.Pid))
	if err != nil {
		return 0, err
	}
	f := strings.Fields(string(stat))
	startTime, err := strconv.ParseInt(f[21], 10, 64)
	return Time(startTime), err
}

func Since(t Time) time.Duration {
	return Now().Sub(t)
}
