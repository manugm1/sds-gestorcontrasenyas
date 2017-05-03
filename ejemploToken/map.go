package main

import (
    "fmt"

)
type FactorDoble struct{
  Token string
}
func main() {
var tokens = make(map[string]FactorDoble)
var t FactorDoble
t.Token="hola"
tokens["a"]=t
fmt.Println(tokens["a"].Token)
}
