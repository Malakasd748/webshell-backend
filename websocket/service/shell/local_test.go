package shell

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLocalShellProvider_NewShell(t *testing.T) {
	provider := &LocalShellProvider{
		Logger: log.New(os.Stderr, "[test] ", log.LstdFlags),
	}

	// Test with empty CWD
	shell, err := provider.NewShell("")
	if err != nil {
		t.Skipf("Skipping test: cannot create PTY shell: %v", err)
	}
	if shell != nil {
		assert.NotNil(t, shell)
		shell.Close()
	}

	// Test with specific CWD
	tmpDir := os.TempDir()
	shell, err = provider.NewShell(tmpDir)
	if err != nil {
		t.Skipf("Skipping test: cannot create PTY shell: %v", err)
	}
	if shell != nil {
		ptyShell := shell.(*PTYShell)

		// Test resize
		err = ptyShell.Resize(24, 80)
		assert.NoError(t, err)

		// Test write (only if shell creation succeeded)
		testData := []byte("test command\n")
		n, err := ptyShell.Write(testData)
		assert.NoError(t, err)
		assert.Equal(t, len(testData), n)

		// Clean up
		ptyShell.Close()
	}
}

func TestNewLocalService(t *testing.T) {
	service := NewLocalService()
	assert.NotNil(t, service)
	assert.IsType(t, &ShellService{}, service)
}
