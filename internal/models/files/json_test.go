package files

import (
	"context"
	"fmt"
	confModule "github.com/MaximMNsk/go-url-shortener/server/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"path/filepath"
	"testing"
)

var Conf confModule.OuterConfig
var ConfErr error

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)

	return nBytes, err
}

func restoreFile(source, dest string) error {
	errRemove := os.Remove(dest)
	if errRemove != nil {
		return errRemove
	}
	_, err := copyFile(source, dest)
	if err != nil {
		return err
	}
	return nil
}

func TestJSONDataSet_Set(t *testing.T) {
	ConfErr = Conf.InitConfig(true)
	type fields struct {
		Link      string
		ShortLink string
		ID        string
	}
	type args struct {
		fileName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "Set",
			fields: fields{
				Link:      "TestLink",
				ShortLink: "TestShortLink",
				ID:        "TestID",
			},
			args: args{fileName: filepath.Join(Conf.Default.LinkFile)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, ConfErr)
			jsonData := &FileStorage{}
			jsonData.Init(tt.fields.Link, tt.fields.ShortLink, tt.fields.ID, false, context.Background(), Conf)
			err := jsonData.Set()
			assert.NoError(t, err)
			require.FileExists(t, filepath.Join(tt.args.fileName))
		})
	}
}

func TestJSONDataGet_Get(t *testing.T) {
	ConfErr = Conf.InitConfig(true)
	type fields struct {
		Link      string
		ShortLink string
		ID        string
	}
	type want FileStorage
	type args struct {
		fileName       string
		sourceFileName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{
			name: "Get",
			fields: fields{
				Link: "TestLink",
			},
			args: args{
				fileName: filepath.Join(Conf.Default.LinkFile),
			},
			want: want(FileStorage{
				Link:      "TestLink",
				ShortLink: "TestShortLink",
				ID:        "TestID",
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.FileExists(t, tt.args.fileName)
			jsonData := FileStorage{}
			jsonData.Init(tt.fields.Link, tt.fields.ShortLink, tt.fields.ID, false, context.Background(), Conf)
			link, _, err := jsonData.Get()
			assert.NoError(t, err)
			assert.EqualValues(t, tt.want.Link, link)
		})
	}
}
