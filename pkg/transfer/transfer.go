package transfer

type Uploader interface {
	Upload(localFile, remoteFile string) error
}

type Downloader interface {
	Download(remoteFile, localFile string) error
}

type Transfer interface {
	Uploader
	Downloader
	ListFile(dir string) chan string
}
