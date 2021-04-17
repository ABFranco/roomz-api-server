package util
import (
  "fmt"
  "math/rand"
)
func RandStr() string {
  // string-length range: [1, 8]
	return fmt.Sprintf("%v", fmt.Sprintf("%v", rand.Intn(10000000)))
}