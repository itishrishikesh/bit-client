package downloader

import (
	"bit-client/utils"
	"testing"
)

func TestDownload(t *testing.T) {
	torrent := utils.GetTestTorrent()
	downloader := New(torrent)
	downloader.Download("debian.iso")
}
