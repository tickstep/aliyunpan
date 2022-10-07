package command

import (
	"fmt"
	"testing"
)

func TestWildcard(t *testing.T) {
	fmt.Println(isIncludeFile("ab0.txt", "ab0.txt"))
	fmt.Println(isIncludeFile("*.zip", "aliyunpan-v0.0.1-darwin-macos-amd64.zip"))
	fmt.Println(isIncludeFile(".*.swp", ".vd.txt.swp"))
	fmt.Println(isIncludeFile("*.swp", ".swp"))
	fmt.Println(isIncludeFile("*.swp", ".1swp"))

	fmt.Println(isIncludeFile("a*b", "ab"))
	fmt.Println(isIncludeFile("a*b", "aab"))
	fmt.Println(isIncludeFile("a*b", "accccccccdb"))
	fmt.Println(isIncludeFile("a?b", "acb"))
	fmt.Println(isIncludeFile("a?b", "accb"))
	fmt.Println(isIncludeFile("a[xyz]b", "axb"))
	fmt.Println(isIncludeFile("ab[0-9].txt", "ab0.txt"))

	fmt.Println(isIncludeFile("a*b/ab[0-9].txt", "acb/ab0.txt"))
	fmt.Println(isIncludeFile("aliyunpan*", "aliyunpan-v0.0.1-darwin-macos-amd64[TNT].zip"))
}
