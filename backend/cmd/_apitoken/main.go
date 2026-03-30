package main
import (
  "fmt"
  "github.com/bigops/platform/internal/pkg/config"
  jwtpkg "github.com/bigops/platform/internal/pkg/jwt"
)
func main() {
  if err := config.Load("config/config.yaml"); err != nil { panic(err) }
  token, err := jwtpkg.GenerateToken(1, "admin")
  if err != nil { panic(err) }
  fmt.Print(token)
}
