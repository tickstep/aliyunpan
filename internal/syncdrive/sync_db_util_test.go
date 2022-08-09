package syncdrive

import (
	"fmt"
	"testing"
)

func TestFormatFilePath(t *testing.T) {
	fmt.Println(FormatFilePath("D:\\-beyond\\p\\9168473.html"))
}

func TestFormatFilePath2(t *testing.T) {
	fmt.Println(FormatFilePath("/my/folder/test.txt"))
}
