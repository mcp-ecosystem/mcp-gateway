package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestProcessFunctions(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	t.Run("SendSignalToPIDFile with empty path", func(t *testing.T) {
		err := SendSignalToPIDFile("", unix.SIGTERM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PID file path is empty")
	})

	t.Run("SendSignalToPIDFile with non-existent file", func(t *testing.T) {
		err := SendSignalToPIDFile("non_existent.pid", unix.SIGTERM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read PID file")
	})

	t.Run("readPIDFile with invalid content", func(t *testing.T) {
		err := os.WriteFile(pidFile, []byte("invalid"), 0644)
		require.NoError(t, err)

		_, err = readPIDFile(pidFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PID format")
	})

	t.Run("readPIDFile with invalid PID value", func(t *testing.T) {
		err := os.WriteFile(pidFile, []byte("0"), 0644)
		require.NoError(t, err)

		_, err = readPIDFile(pidFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid PID value")
	})

	t.Run("readPIDFile with valid PID", func(t *testing.T) {
		pid := os.Getpid()
		err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644)
		require.NoError(t, err)

		readPID, err := readPIDFile(pidFile)
		assert.NoError(t, err)
		assert.Equal(t, pid, readPID)
	})

	t.Run("signalProcess with non-existent PID", func(t *testing.T) {
		err := signalProcess(999999, unix.SIGTERM)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send signal")
	})
}
