package shell

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvCWD(t *testing.T) {
	// Save original environment and restore it after test
	originalEnv := os.Getenv(envName)
	defer os.Setenv(envName, originalEnv)

	tests := []struct {
		name     string
		envValue string
		setup    func()
		want     string
	}{
		{
			name:     "with valid environment variable",
			envValue: os.TempDir(),
			want:     os.TempDir(),
		},
		{
			name:     "with invalid path",
			envValue: "/non/existent/path",
			want:     mustGetHomeDir(t),
		},
		{
			name:     "with empty environment variable",
			envValue: "",
			want:     mustGetHomeDir(t),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			os.Setenv(envName, tt.envValue)
			got := getEnvCWD()
			assert.Equal(t, tt.want, got)
		})
	}
}

// Helper function to get home directory or fail test
func mustGetHomeDir(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	return home
}
