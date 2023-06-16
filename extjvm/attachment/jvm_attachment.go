package attachment

import (
  "bytes"
  "context"
  "github.com/rs/zerolog/log"
  "github.com/steadybit/extension-jvm/extjvm/jvm"
  "github.com/steadybit/extension-jvm/extjvm/utils"
  "os"
  "os/user"
  "path/filepath"
  "runtime"
  "strconv"
  "time"
)

func externalAttach(jvm *jvm.JavaVm, agentJar string, initJar string, agentHTTPPort int, host string, addNsEnter bool, pid string, hostpid string) bool {
	initJarAbsPath, err := filepath.Abs(initJar)
	if err != nil {
		log.Error().Err(err).Msgf("Could not determine absolute path of init jar %s", initJar)
		return false
	}
	agentJarAbsPath, err := filepath.Abs(agentJar)
	if err != nil {
		log.Error().Err(err).Msgf("Could not determine absolute path of agent jar %s", agentJar)
		return false
	}
	attachCommand := []string{
		getJavaExecutable(jvm),
		"-Xms16m",
		"-Xmx16m",
		"-XX:+UseSerialGC",
		"-XX:+PerfDisableSharedMem",
		"-Dsun.tools.attach.attachTimeout=30000",
		"-Dsteadybit.agent.disable-jvm-attachment",
		"-jar",
		initJarAbsPath,
		"pid=" + pid,
		"hostpid=" + hostpid,
		"host=" + host,
		"port=" + strconv.Itoa(agentHTTPPort),
		"agentJar=" + agentJarAbsPath,
	}

  if addNsEnter {
    nsEnterCommand := []string{"nsenter", "-t", strconv.Itoa(int(jvm.Pid))}
    nsEnterCommand = append(nsEnterCommand, "-m", "-p", "--")
    attachCommand = append(nsEnterCommand, attachCommand...)
  }

	if needsUserSwitch(jvm) {
		attachCommand = addUserIdAndGroupId(jvm, attachCommand)
	}

	log.Trace().Msgf("Executing attach command on host: %s", attachCommand)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()
  log.Info().Msgf("Command: %s", attachCommand)
	cmd := utils.RootCommandContext(ctx, attachCommand[0], attachCommand[1:]...)
  var outb, errb bytes.Buffer
  cmd.Stdout = &outb
  cmd.Stderr = &errb
  err = cmd.Run()
  log.Info().Msgf("Attach command output: %s", outb.String())
  log.Info().Msgf("Attach command error: %s", errb.String())
	if err != nil {
		log.Error().Err(err).Msgf("Error attaching to JVM %+v: %s", jvm, err)
		return false
	}
	return true
}

func addUserIdAndGroupId(vm *jvm.JavaVm, attachCommand []string) []string {
	if vm.GroupId != "" && vm.UserId != "" {
		return append(attachCommand, "uid="+vm.UserId, "gid="+vm.GroupId)
	}
	return attachCommand
}

func needsUserSwitch(jvm *jvm.JavaVm) bool {
	current, err := user.Current()
	if err != nil {
		log.Warn().Err(err).Msg("Could not determine current user")
		return false
	}
	return !(jvm.UserId == current.Uid && jvm.GroupId == current.Gid)
}

func getJavaExecutable(jvm *jvm.JavaVm) string {
	if jvm.Path != "" && (isExecAny(jvm.Path)) {
		return jvm.Path
	} else {
    if runtime.GOOS == "windows" {
      return "java.exe"
    }
		return "java"
	}
}

func isExecAny(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.Mode()&0111 != 0
}
