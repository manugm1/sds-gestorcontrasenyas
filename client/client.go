package main

import (
    "fmt"
    "io/ioutil"
    "net"
    "os"
    "encoding/json"
)


type Usuario struct {
    Id         int
    User    string
    Contraseña string
}


func main() {
  res1D := &Usuario{
      Id:   1,
      User: "yo",
      Contraseña: "yo"}
  res1B, _ := json.Marshal(res1D)
  fmt.Println(string(res1B))



    if len(os.Args) != 2 {
        fmt.Fprintf(os.Stderr, "Usage: %s host:port ", os.Args[0])
        os.Exit(1)
    }
    service := os.Args[1]
    tcpAddr, err := net.ResolveTCPAddr("tcp4", service)
    checkError(err)
    conn, err := net.DialTCP("tcp", nil, tcpAddr)
    checkError(err)
    _, err = conn.Write([]byte(string(res1B)))
    checkError(err)
    result, err := ioutil.ReadAll(conn)
    checkError(err)
    fmt.Println(string(result))
    os.Exit(0)
}
func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
        os.Exit(1)
    }
}
