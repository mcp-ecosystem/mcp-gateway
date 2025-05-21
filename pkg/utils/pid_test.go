package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIDManager(t *testing.T) {
	tmpDir := t.TempDir()
	pidFile := filepath.Join(tmpDir, "test.pid")

	t.Run("NewPIDManager", func(t *testing.T) {
		manager := NewPIDManager(pidFile)
		assert.Equal(t, pidFile, manager.GetPIDFile())
	})

	t.Run("NewPIDManagerFromConfig", func(t *testing.T) {
		manager := NewPIDManagerFromConfig(pidFile)
		assert.Equal(t, pidFile, manager.GetPIDFile())
	})

	t.Run("WritePID", func(t *testing.T) {
		manager := NewPIDManager(pidFile)
		err := manager.WritePID()
		require.NoError(t, err)

		content, err := os.ReadFile(pidFile)
		require.NoError(t, err)

		pidStr := strings.TrimSpace(string(content))
		pid, err := strconv.Atoi(pidStr)
		require.NoError(t, err)
		assert.Equal(t, os.Getpid(), pid)
	})

	t.Run("RemovePID", func(t *testing.T) {
		manager := NewPIDManager(pidFile)
		err := manager.WritePID()
		require.NoError(t, err)

		err = manager.RemovePID()
		require.NoError(t, err)

		_, err = os.Stat(pidFile)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("WritePID with non-existent directory", func(t *testing.T) {
		manager := NewPIDManager(filepath.Join(tmpDir, "subdir", "test.pid"))
		err := manager.WritePID()
		require.NoError(t, err)

		_, err = os.Stat(manager.GetPIDFile())
		require.NoError(t, err)
	})
}
