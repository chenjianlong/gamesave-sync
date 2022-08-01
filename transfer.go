package main

type Uploader interface {
	upload(localFile, remoteFile string) error
}

type Downloader interface {
	download(remoteFile, localFile string) error
}

type Transfer interface {
	Uploader
	Downloader
}