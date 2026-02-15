package updater

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// replaceCurrentBinary installs sourcePath over the running executable.
// Returns appliedNow=false on Windows where replacement is staged and applied
// after process exit.
func replaceCurrentBinary(sourcePath string) (appliedNow bool, err error) {
	execPath, err := currentExecutablePath()
	if err != nil {
		return false, err
	}

	if runtime.GOOS == "windows" {
		if err := scheduleWindowsReplacement(execPath, sourcePath); err != nil {
			return false, err
		}
		return false, nil
	}

	if err := replaceBinaryNow(execPath, sourcePath); err != nil {
		return false, err
	}
	finishReplace(execPath)
	return true, nil
}

func currentExecutablePath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	return execPath, nil
}

func replaceBinaryNow(execPath, sourcePath string) error {
	oldPath := execPath + ".old"
	if err := os.Rename(execPath, oldPath); err != nil {
		return fmt.Errorf("move current binary: %w", err)
	}

	if err := copyFile(sourcePath, execPath); err != nil {
		_ = os.Rename(oldPath, execPath) // best-effort rollback
		return fmt.Errorf("install new binary: %w", err)
	}
	_ = os.Remove(oldPath)
	return nil
}

func finishReplace(execPath string) {
	if runtime.GOOS != "windows" {
		_ = os.Chmod(execPath, 0o755)
	}
	binDir := filepath.Dir(execPath)
	baseName := filepath.Base(execPath)
	recreateLinks(binDir, baseName)
}

func scheduleWindowsReplacement(execPath, sourcePath string) error {
	stagePath := execPath + ".new"
	_ = os.Remove(stagePath)
	if err := copyFile(sourcePath, stagePath); err != nil {
		return fmt.Errorf("prepare staged binary: %w", err)
	}

	scriptPath := filepath.Join(os.TempDir(), fmt.Sprintf("firecommit-replace-%d.cmd", time.Now().UnixNano()))
	script := windowsReplaceScript(execPath, stagePath)
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		return fmt.Errorf("write replace script: %w", err)
	}

	if err := exec.Command("cmd", "/C", "start", "", "/B", scriptPath).Run(); err != nil {
		return fmt.Errorf("launch replace script: %w", err)
	}
	return nil
}

func windowsReplaceScript(execPath, stagePath string) string {
	target := escapeBatchValue(execPath)
	staged := escapeBatchValue(stagePath)
	old := escapeBatchValue(execPath + ".old")
	binDir := escapeBatchValue(filepath.Dir(execPath))

	return fmt.Sprintf(`@echo off
setlocal
set "TARGET=%s"
set "STAGED=%s"
set "OLD=%s"
set "BIN_DIR=%s"

:wait_target
move /Y "%%TARGET%%" "%%OLD%%" >nul 2>&1
if errorlevel 1 (
  timeout /T 1 /NOBREAK >nul
  goto wait_target
)

move /Y "%%STAGED%%" "%%TARGET%%" >nul 2>&1
if errorlevel 1 (
  move /Y "%%OLD%%" "%%TARGET%%" >nul 2>&1
  exit /b 1
)

copy /Y "%%TARGET%%" "%%BIN_DIR%%\fcmt.exe" >nul 2>&1
copy /Y "%%TARGET%%" "%%BIN_DIR%%\git-fire-commit.exe" >nul 2>&1
del /F /Q "%%OLD%%" >nul 2>&1
del /F /Q "%%~f0" >nul 2>&1
`, target, staged, old, binDir)
}

func escapeBatchValue(s string) string {
	return strings.ReplaceAll(s, "%", "%%")
}
