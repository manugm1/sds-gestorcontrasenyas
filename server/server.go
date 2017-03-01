package main

import (
    "fmt"
    "net"
    "os"
    "time"
    //"bufio"
)

func main() {
    service := ":8080"
    tcpAddr, err := net.ResolveTCPAddr("tcp4", service)

    checkError(err)
    listener, err := net.ListenTCP("tcp", tcpAddr)
    //fmt.Print("Enter text: " ,listener)
    checkError(err)
    for {
        conn, err := listener.Accept()
        if err != nil {
            continue
        }

        go handleClient(conn)
    }
}

func handleClient(conn net.Conn) {
  request := make([]byte, 128) // set maximum request length to 128B to prevent flood based attacks
    defer conn.Close()

    /*// will listen for message to process ending in newline (\n)
    message, _ := bufio.NewReader(conn).ReadString('\n')
    // output message received
    fmt.Print("Message Received:", string(message))
    // sample process for string received*/

    for {
        read_len, err := conn.Read(request)

        /*if err != nil {
            fmt.Println(err)
            break
        }*/
        checkError(err)
        if read_len == 0 {
            break // connection already closed by client
        }else if read_len>0{
          s := string(request[:read_len])
              fmt.Print("Message Received:", s)

          }else {
            daytime := time.Now().String()
            conn.Write([]byte(daytime))
        }
}
}
func checkError(err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
        os.Exit(1)
    }
}
