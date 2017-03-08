package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Usuario struct {
	Id         int
	User       string
	Contraseña string
}

/**
* Muestra el primer menú: Login, Registro y Salir
 */
func primerMenu() {
	for {
		fmt.Println("-------Elige opción [1-2] o 'q' para salir-------")
		fmt.Println("[1] Login")
		fmt.Println("[2] Registro")
		fmt.Println("[q] Salir")
		reader := bufio.NewReader(os.Stdin)
		opcionElegida, error := reader.ReadString('\n')
		checkError(error)
		opcionElegida = strings.TrimSpace(strings.Replace(opcionElegida, " ", "", -1)) // quitamos los espacios
		switch opcionElegida {
		case "1":
			fmt.Println("Se ha elegido login")
			login()
			break
		case "2":
			fmt.Println("Se ha elegido registro")
			registro()
			break
		case "q", "Q":
			fmt.Println("Se ha elegido SALIR")
			os.Exit(1) //finalizamos el programa
		}
	}
}

/**
* Operación de login sobre el servidor
 */
func login() {

}

/**
* Operación de registro sobre el servidor
 */
func registro() {

}

func main() {

	fmt.Println("-------GESTOR DE CONTRASEÑAS-------")

	primerMenu()

	/*
		res1D := &Usuario{
			Id:         1,
			User:       "yo",
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
		os.Exit(0)*/
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
