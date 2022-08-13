package transfer

import (
	"github.com/jlaffaye/ftp"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

type FTPTransfer struct {
	conn     *ftp.ServerConn
	subDir string
}

func NewFTPTransfer(addr, user, password, subDir string) (Transfer, error) {
	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}

	if err = conn.Login(user, password); err != nil {
		return nil, err
	}

	transfer := new(FTPTransfer)
	transfer.conn = conn
	transfer.subDir = subDir
	return transfer, nil
}

func (t *FTPTransfer) Upload(localFile, remoteFile string) error {
	fs, err := os.Open(localFile)
	if err != nil {
		return err
	}
	defer fs.Close()

	remoteFile = path.Join(t.subDir, remoteFile)
	err = t.conn.Stor(remoteFile, fs)
	if err != nil && err.Error() == "550 Couldn't open the file or directory" {
		elements := strings.Split(remoteFile, "/")
		for i := 1; i < len(elements); i += 1 {
			remoteDir := path.Join(elements[:i]...)
			if err = t.conn.MakeDir(remoteDir); err != nil {
				return err
			}
		}

		err = t.conn.Stor(remoteFile, fs)
	}

	return err
}

func (t *FTPTransfer) Download(remoteFile, localFile string) error {
	fs, err := os.Create(localFile)
	if err != nil {
		return err
	}

	remoteFile = path.Join(t.subDir, remoteFile)
	resp, err := t.conn.Retr(remoteFile)
	if err != nil {
		return err
	}

	defer resp.Close()
	_, err = io.Copy(fs, resp)
	return err
}

func (t *FTPTransfer) ListFile(dir string) chan string {
	resultCh := make(chan string)
	go func() {
		dir = path.Join(t.subDir, dir)
		entries, err := t.conn.List(dir)
		if err != nil {
			close(resultCh)
			return
		}

		for _, entry := range entries {
			if entry.Type != ftp.EntryTypeFile {
				continue
			}

			resultCh <- entry.Name
		}
		close(resultCh)
	}()
	return resultCh
}
