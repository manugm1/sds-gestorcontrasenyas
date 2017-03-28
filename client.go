package main

import (
	"bufio"
	"encoding/json"
	"encoding/base64"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"net/http/cookiejar"
)

// respuesta del servidor
type Resp struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
}

type Usuario struct{
	Email string
	Password string
}

var SesionIniciada = false //para probar el inicio de sesión y que salga el menú principal.
														//De momento solo se pone a true cuando hacemos login
														//Al cerrar sesión se pone a false de nuevo
														//Esto habría que implementarlo con sesiones reales
var cookie []*http.Cookie
var usuarioActual Usuario

/**
* [1] Operación de registro sobre el servidor
 */
func registro() {
	//Introducir usuario y contraseña
	fmt.Println("Introduce email: ")
	email := leerStringConsola()

	fmt.Println("Introduce contraseña: ")
	password := leerStringConsola()

	//Generamos los parámetros a enviar al servidor
	parametros := url.Values{}
	parametros.Set("opcion", "1")
	//Pasamos el parámetro a la estructura Usuario
	usuario := Usuario{Email: email, Password: password}
	parametros.Set("usuario", codificarStructToJSONBase64(usuario))

	//Pasar parámetros al servidor
	cadenaJSON := comunicarServidor(parametros)

	var respuesta Resp
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &respuesta)
	checkError(error)

	//Mostramos sí o sí lo que nos devuelve como respuesta el servidor
	fmt.Println(respuesta.Msg)
}

/**
* [2] Operación de login sobre el servidor
* Prácticamente igual al registro pero tiene en cuenta la sesión
 */
func login(){
	//Introducir usuario y contraseña
	fmt.Println("Introduce email: ")
	email := leerStringConsola()

	fmt.Println("Introduce contraseña: ")
	password := leerStringConsola()

	//Generamos los parámetros a enviar al servidor
	parametros := url.Values{}
	parametros.Set("opcion", "2")
	//Pasamos el parámetro a la estructura Usuario
	usuario := Usuario{Email: email, Password: password}
	parametros.Set("usuario", codificarStructToJSONBase64(usuario))

	//Pasar parámetros al servidor
	cadenaJSON := comunicarServidor(parametros)

	var respuesta Resp
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &respuesta)
	checkError(error)
	//Manejamos la respuesta de login del servidor
	if respuesta.Ok {
		usuarioActual = usuario
	}
	//Mostramos sí o sí lo que nos devuelve como respuesta el servidor
	fmt.Println(respuesta.Msg)
}

func esLogueado() (bool){
	devolver := false
	//Lo primero: asegurarse que el usuario actual tiene valor
	if usuarioActual != (Usuario{}) {
		//Generamos los parámetros a enviar al servidor
		parametros := url.Values{}
		parametros.Set("opcion", "3")
		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)

		var respuesta Resp
		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)
		devolver = respuesta.Ok
	}
	return devolver
}

func logout(){
	if usuarioActual != (Usuario{}) {
		//Generamos los parámetros a enviar al servidor
		parametros := url.Values{}
		parametros.Set("opcion", "4")
		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)
		var respuesta Resp
		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)
		fmt.Println("resultado del logout: "+string(cadenaJSON))
		//dar de baja el usuarioactual y la cookie
	}
}

func listarEntradas(){

}

func anyadirEntrada(){

}

func editarEntrada(){

}

func borrarEntrada(){

}

func main() {
	fmt.Println("-------GESTOR DE CONTRASEÑAS-------")
	menu()
}

/**
* Maneja el menú, mostrando el menú inicial si NO hay sesión iniciada
* y menú principal si la sesión SÍ está iniciada
 */
func menu() {
	//Mostrar menú principal del usuario
	for{
		//Si no está logueado, mostramos login/registro
		if !esLogueado() {
			menuInicio()
		} else{ //Mostrar menú principal con todas las opciones
			menuPrincipal()
		}
	}
}

func menuInicio(){
		fmt.Println("-------Elige opción [1-2] o 'q' para salir-------")
		fmt.Println("[1] Registro")
		fmt.Println("[2] Login")
		fmt.Println("[q] Salir")
		opcionElegida := leerStringConsola()

		switch opcionElegida {
		case "1":
			fmt.Println("Se ha elegido registro")
			registro()
			break
		case "2":
			fmt.Println("Se ha elegido login")
			login()
			break
		case "q", "Q":
			fmt.Println("Se ha elegido SALIR")
			os.Exit(1) //finalizamos el programa
		}
}

func menuPrincipal(){
		fmt.Println("-------¡Bienvenido!-------")
		fmt.Println("-------Elige opción [1-4] o 'q' para cerrar sesión-------")
		fmt.Println("[1] Listar entradas")
		fmt.Println("[2] Añadir entrada")
		fmt.Println("[3] Editar entrada")
		fmt.Println("[4] Borrar entrada")
		fmt.Println("[q] Cerrar sesión")
		opcionElegida := leerStringConsola()

		switch opcionElegida {
		case "1":
			fmt.Println("Se ha elegido listar entradas")
			listarEntradas()
			break
		case "2":
			fmt.Println("Se ha elegido añadir entrada")
			anyadirEntrada()
			break
		case "3":
			fmt.Println("Se ha elegido editar entrada")
			editarEntrada()
			break
		case "4":
			fmt.Println("Se ha elegido borrar entrada")
			borrarEntrada()
			break
		case "q", "Q":
			fmt.Println("Se ha elegido cerrar sesión")
			logout()
			menuInicio() //volvemos al primer menú, será en éste donde se pueda
									 //salir del programa
			break
		}
}

/**
* Lee un string por consola
*/
func leerStringConsola()(string){
	reader := bufio.NewReader(os.Stdin)
	lectura, error := reader.ReadString('\n')
	checkError(error)
	return strings.TrimSpace(strings.Replace(lectura, " ", "", -1)) // quitamos los espacios
}

/**
* Método para comunicar con el servidor pasándole una serie de parámetros (url.Values)
* Devuelve la respuesta del body codificado en bytes, para que sea el propio
* método invocador el que parsee como desee la respuesta
*/
func comunicarServidor(parametros url.Values)([]byte){
	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse("https://localhost:10443")
	jar.SetCookies(u, cookie)
	fmt.Println("ANTES RESPUESTA------")
	for k, v := range cookie {
				fmt.Println("key:", k, "value:", v)
	}
	transporte := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cliente := &http.Client{Transport: transporte, Jar: jar}
	//Pasamos los parámetros codificados a base 64 para enviarlos de forma
	//segura (sin caracteres problemáticos añadidos)
	peticion, error := cliente.PostForm("https://localhost:10443", parametros)
	checkError(error)

	//Pasamos el cuerpo de la respuesta del servidor a bytes para devolverlo
	//El método que invoca a este método ya verá si lo pasa a string o a struct
	respuesta, error := ioutil.ReadAll(peticion.Body)
	checkError(error)
	//Paso más, pasamos a JSON simple, decodificamos JSON base 64
	respuesta = decodificarJSONBase64ToJSON(string(respuesta))

	fmt.Println("DESPUES RESPUESTA------")
	//Debug, mostramos las cabeceras header que devuelve el servidor
	for k, v := range peticion.Cookies() {
				fmt.Println("key:", k, "value:", v)
	}
	cookie = peticion.Cookies()
	return respuesta
}

/**
* Codificamos en JSON una estructura cualquiera y
* devolvemos codificado el JSON en base64
*/
func codificarStructToJSONBase64(estructura interface{})(string){
	//codificamos en JSON
	respJSON, error := json.Marshal(&estructura)
	checkError(error)
	//codificamos en base64 para que no dé problemas al enviar al servidor
	respuesta := base64.StdEncoding.EncodeToString(respJSON)
	return respuesta
}

/**
* Decodifica un json en base 64 (viene del cliente así)
* y lo pasa a []byte que es un json simple que hay que des-serializar
*/
func decodificarJSONBase64ToJSON(cadenaCodificada string)([]byte){
	//Decodificamos el base64
	cadena, error := base64.StdEncoding.DecodeString(cadenaCodificada)
	checkError(error)

	//Pasamos a []byte de JSON
	respuesta := []byte(cadena)
	return respuesta
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
