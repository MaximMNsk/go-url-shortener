package files

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"testing"
)

func copy(src, dst string) (int64, error) {
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
	_, err := copy(source, dest)
	if err != nil {
		return err
	}
	return nil
}

func TestJSONDataSet_Set(t *testing.T) {
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
			args: args{fileName: "./test.json"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonData := &JSONDataSet{
				Link:      tt.fields.Link,
				ShortLink: tt.fields.ShortLink,
				ID:        tt.fields.ID,
			}
			jsonData.Set(tt.args.fileName)
			require.FileExists(t, tt.args.fileName)
		})
	}
}

func TestJSONDataGet_Get(t *testing.T) {
	type fields struct {
		Link      string
		ShortLink string
		ID        string
	}
	type want JSONDataGet
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
				fileName:       "./test.json",
				sourceFileName: "./test_source.json",
			},
			want: want(JSONDataGet{
				Link:      "TestLink",
				ShortLink: "TestShortLink",
				ID:        "TestID",
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.FileExists(t, tt.args.fileName)
			defer func(source string, dest string) {
				err := restoreFile(source, dest)
				if err != nil {
					t.Error(err)
				}
			}(tt.args.sourceFileName, tt.args.fileName)
			jsonData := JSONDataGet{
				Link:      tt.fields.Link,
				ShortLink: tt.fields.ShortLink,
				ID:        tt.fields.ID,
			}
			jsonData.Get(tt.args.fileName)
			assert.EqualValues(t, tt.want, jsonData)
		})
	}
}
