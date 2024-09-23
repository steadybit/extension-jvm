package container

import (
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/steadybit/extension-kit/extutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	detectedRuntime       = autoDetectContainerRuntime()
	regexSchedulerThreads = regexp.MustCompile(`^.+ \((\d+), #threads: \d+\)`)
	regexContainerId      = regexp.MustCompile(`([a-f0-9]{64})`)
)

func autoDetectContainerRuntime() runtime {
	for _, r := range allRuntimes {
		if _, err := os.Stat(r.defaultSocket()); err == nil {
			return r
		}
	}
	return ""
}

func GetRuncRoot() string {
	return detectedRuntime.defaultRuncRoot()
}

func GetContainerIdForProcess(process *process.Process) string {
	p := filepath.Join("/proc", strconv.Itoa(int(process.Pid)), "cgroup")
	cgroup, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return regexContainerId.FindString(string(cgroup))
}

func FindMappedPidInContainer(hostPid int32) int32 {
	pid := findNamespacePid(hostPid)
	if pid > 0 {
		log.Debug().Msgf("Found Host PID %d is %d in container via proc/status", hostPid, pid)
		return pid
	}

	pid = readPidFromSchedulerDebug(hostPid)
	if pid > 0 {
		log.Debug().Msgf("Found Host PID %d is %d in container via proc/sched", hostPid, pid)
		return pid
	}

	log.Debug().Msgf("Could not find container PID for Host PID %d", hostPid)
	return 0
}

func readPidFromSchedulerDebug(hostPid int32) int32 {
	schedPath := filepath.Join("/proc", strconv.Itoa(int(hostPid)), "sched")
	content, err := os.ReadFile(schedPath)
	if err != nil {
		log.Trace().Msgf("Could not read %s: %s", schedPath, err)
		return 0
	}
	submatch := regexSchedulerThreads.FindStringSubmatch(string(content))
	if len(submatch) < 2 {
		return 0
	}
	return extutil.ToInt32(submatch[1])
}

func findNamespacePid(hostPid int32) int32 {
	nsPids := readNsPids(hostPid)
	for i, pid := range nsPids {
		if pid == hostPid {
			if i < len(nsPids)-1 {
				return nsPids[i+1]
			} else {
				return pid
			}
		}
	}
	return 0
}

func readNsPids(hostPid int32) []int32 {
	nsPidsPath := filepath.Join("/proc", strconv.Itoa(int(hostPid)), "status")
	file, err := os.ReadFile(nsPidsPath)
	if err != nil {
		return nil
	}
	nsPids := strings.Split(string(file), "\n")
	for _, nsPid := range nsPids {
		if strings.HasPrefix(nsPid, "NSpid:\t") {
			tokens := strings.Split(nsPid[len("NSpid:\t"):], "\t")
			pids := make([]int32, len(tokens))
			for _, pid := range tokens {
				pids = append(pids, extutil.ToInt32(pid))
			}
			return pids
		}
	}
	return nil
}
