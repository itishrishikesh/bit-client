package utils

import (
	"bit-client/torrent"
	"os"
)

func GetTestFile() *torrent.TorrentFile {
	reader, _ := os.Open("../.nocode/sample.torrent")
	torrent, _ := torrent.Read(reader)
	file, _ := torrent.ToTorrenFile()
	return file
}

func GetTestTorrent() *torrent.Torrent {
	reader, _ := os.Open("../.nocode/sample.torrent")
	torrent, _ := torrent.Read(reader)
	return torrent
}
