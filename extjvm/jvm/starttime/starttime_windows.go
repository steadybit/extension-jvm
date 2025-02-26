package starttime

import (
	"github.com/shirou/gopsutil/v4/process"
	gotime "time"
)

type Time gotime.Time

func Now() Time {
	return Time(gotime.Now())
}

func (t Time) Sub(o Time) gotime.Duration {
	return gotime.Time(t).Sub(gotime.Time(o))
}

func ForProcess(p *process.Process) (Time, error) {
	t, e := p.CreateTime()
	return Time(gotime.UnixMilli(t)), e
}

func Since(t Time) gotime.Duration {
	return Now().Sub(t)
}
