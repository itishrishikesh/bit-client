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

func TestCalculateBoundsForPiece(t *testing.T) {
	mock := &Downloader{PieceLength: 100, Length: 999}
	begin, end := mock.CalculateBoundsForPiece(9)
	if (end - begin) != 99 {
		t.Fail()
	}
}
