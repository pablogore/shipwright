package main

import (
	"fmt"

	"github.com/google/uuid"
)

func main() {
	if _, err := fmt.Println(uuid.NewString()); err != nil {
		panic(err)
	}
}
