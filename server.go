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
	checkError(err)

	//Escuchamos peticiones de los clientes
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue //Ignoramos el manejador si la petición no es válida
		}
		//Llamamos al manejador de cada cliente
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	request := make([]byte, 128) // set maximum request length to 128B to prevent flood based attacks
	defer conn.Close()           //cerramos la conexión sea cual sea el resultado
	for {
		read_len, err := conn.Read(request)
		checkError(err)
		//Si la lectura ha acabado
		if read_len == 0 {
			break // connection already closed by client
		} else if read_len > 0 {
			s := string(request[:read_len])
			fmt.Print("Message Received: ", s)
		} else {
			daytime := time.Now().String()
			conn.Write([]byte(daytime))
		}
	}
}

/**
* Función que chequea los errores y muestra por pantalla en caso de haber alguno
 */
func checkError(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %s", err.Error())
		os.Exit(1)
	}
}
