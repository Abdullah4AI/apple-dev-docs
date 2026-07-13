package install

import (
	"os"
	"os/exec"
	"strings"
)

func defaultStartSkillsCheckWorker(spec skillsCheckWorkerSpec) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	return startDetachedSkillsCheckProcess(
		executable,
		nil,
		skillsCheckWorkerEnvironment(os.Environ(), spec),
	)
}

func startDetachedSkillsCheckProcess(executable string, args, env []string) error {
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	cmd := exec.Command(executable, args...)
	cmd.Env = env
	cmd.Stdin = devNull
	cmd.Stdout = devNull
	cmd.Stderr = devNull
	configureDetachedSkillsCheckProcess(cmd)

	if err := cmd.Start(); err != nil {
		return err
	}
	// The worker owns its lifecycle after Start. Release prevents the foreground
	// process from retaining a wait obligation or zombie bookkeeping.
	_ = cmd.Process.Release()
	return nil
}

func skillsCheckWorkerEnvironment(base []string, spec skillsCheckWorkerSpec) []string {
	workerKeys := map[string]struct{}{
		skillsWorkerEnvVar:      {},
		skillsWorkerCacheEnvVar: {},
		skillsWorkerLockEnvVar:  {},
		skillsWorkerTokenEnvVar: {},
	}
	env := make([]string, 0, len(base)+len(workerKeys))
	for _, entry := range base {
		key, _, _ := strings.Cut(entry, "=")
		if _, isWorkerKey := workerKeys[key]; !isWorkerKey {
			env = append(env, entry)
		}
	}
	return append(
		env,
		skillsWorkerEnvVar+"=1",
		skillsWorkerCacheEnvVar+"="+spec.cachePath,
		skillsWorkerLockEnvVar+"="+spec.lockPath,
		skillsWorkerTokenEnvVar+"="+spec.token,
	)
}
