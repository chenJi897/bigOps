package main

import (
  "fmt"
  "time"
  a "github.com/bigops/platform/internal/agent"
)

func main() {
  c := a.NewMetricsCollector()
  m1 := c.Collect()
  fmt.Printf("first=%+v\n", m1)
  time.Sleep(1200 * time.Millisecond)
  m2 := c.Collect()
  fmt.Printf("second=%+v\n", m2)
}
