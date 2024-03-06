package main

import (
	"bit-client/downloader"
	"bit-client/torrent"
	"flag"
	"log"
	"os"
)

func main() {
	torrentPath := flag.String("in", "./.nocode/sample.torrent", "torrent file")
	outputPath := flag.String("out", "./", "output path")
	reader, _ := os.Open(*torrentPath)
	t, err := torrent.Read(reader)
	if err != nil {
		log.Fatal("Invalid torrent file.", err)
	}
	d := downloader.New(t)
	d.Download(*outputPath)
}
