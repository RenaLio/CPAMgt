package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	rootCtx := context.Background()
	f1 := context.WithValue(rootCtx, "f1", 123)
	f2 := context.WithValue(f1, "f2", 456)
	f3 := context.WithValue(f2, "f1", 789)
	_ = f3
	fmt.Println(f3.Value("f1"))
	fmt.Println(time.Now().Format(time.RFC3339))
}
