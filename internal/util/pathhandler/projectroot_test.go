package pathhandler

import (
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestProjectRoot(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr bool
	}{
		{
			name:    "Path to root of project tester",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ProjectRoot()
			if (err != nil) != tt.wantErr {
				t.Errorf("ProjectRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			require.FileExists(t, filepath.Join(got, "go.mod"))
		})
	}
}
