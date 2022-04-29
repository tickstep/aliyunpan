package plugins

import (
	"fmt"
	"testing"
)

func TestDeleteLocalFile(t *testing.T) {
	fmt.Println(DeleteLocalFile("/Volumes/Downloads/dev/upload/2"))
}
