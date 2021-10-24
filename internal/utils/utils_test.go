package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestConvertTime(t *testing.T) {
	seconds := time.Duration(50) * time.Second
	fmt.Println(ConvertTime(seconds))

	seconds = time.Duration(150) * time.Second
	fmt.Println(ConvertTime(seconds))

	seconds = time.Duration(3600) * time.Second
	fmt.Println(ConvertTime(seconds))

	seconds = time.Duration(1246852) * time.Second
	fmt.Println(ConvertTime(seconds))
}
