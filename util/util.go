// The util package will host all common utility functions used across all
// packages in the RAS.
package util

import (
  "fmt"
  "math/rand"
)


// RandStr returns a random string of length n where 1 <= n <= 8.
func RandStr() string {
	return fmt.Sprintf("%v", fmt.Sprintf("%v", rand.Intn(10000000)))
}