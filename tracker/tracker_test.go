package tracker

import (
	"bit-client/utils"
	"testing"
)

func TestAnnounce(t *testing.T) {
	file := utils.GetTestFile()
	url, _ := file.TrackerUrl([20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}, 3000)
	peers := Announce(url)
	if len(peers) == 0 {
		t.Fatal("failed to list peers")
	}
}
