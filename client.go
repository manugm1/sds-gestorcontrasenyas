package main

import (
	"bufio"
	"encoding/json"
	"encoding/base64"
	"crypto/tls"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	X "math/rand"
	"crypto/sha512"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"strconv"

)
// respuesta despues de comprobar si el usuario esta en la base de datos
type RespLogin struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	//Dato Token // el token
	Pin string
}

// respuesta del servidor
type Resp struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	Dato Token //el token
	Pin string
}

// respuesta del servidor con peticiones sobre entradas
type RespEntrada struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	Entradas map[int]Entrada
}

/**
* Contraseña usuario: Hasheado 256 bits con clave maestra.
* Se envía al servidor para calcular la salt y hacer un scrypt
*/
type Usuario struct{
	Email string
	Password string
}
type Token struct{
  Dato2 string
}


type UsuarioMod struct{
	Email string
	Password string
	//Salt string
	Entradas map[int] Entrada
}


/**
* Contraseña entrada: cifrado con AES/CTR desde el cliente.
*/
type Entrada struct {
    Login string
    Password string
    Web string
    Descripcion string
}
var token Token
var claveMaestra []byte //clave maestra generada a partir de la contraseña
												//del usuario, que servirá para cifrar y descifrar las contraseñas
												//de las cuentas
var usuarioActual Usuario //usuario actual (logueado)
var entradas = make(map[int]Entrada)//entradas
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var lettersNumbers = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
var pinRecibido string
func randLetter(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = letters[X.Intn(len(letters))]
    }
    return string(b)
}
func randLettersNumbers(n int) string {
    b := make([]rune, n)
    for i := range b {
        b[i] = lettersNumbers[X.Intn(len(lettersNumbers))]
    }
    return string(b)
}
 func generarContrasenyaAleatoria() string{
	 var numeroCaracteres string
	 var password string
	 var respuestaNumeros string
	//pREGUNTAS
	fmt.Println("¿Nº de caracteres?")
	numeroCaracteres  = leerStringConsola()
	for{
	fmt.Println("¿Añadir números? S/N")
	respuestaNumeros  = leerStringConsola()
	 if strings.EqualFold(respuestaNumeros, "s") || strings.EqualFold(respuestaNumeros, "n"){
		 break
	 }else{

		 fmt.Println("Hay que poner letra s/S o n/N")
	 }
  }
	if strings.EqualFold(respuestaNumeros, "n"){
		i, error := strconv.Atoi(numeroCaracteres)
		checkError(error)
		password = randLetter(i)
	}else if strings.EqualFold(respuestaNumeros, "s"){
		i, error := strconv.Atoi(numeroCaracteres)
		checkError(error)
		password = randLettersNumbers(i)
	}
	///... LO QUE SE TE OCURRA
	//GENERAR CONTRASEÑA ALEATORIA SEGÚN LO QUE HA ELEGIDO EL USUARIO
	return password
} //que devuelva la contraseña aleatoria

/**
* [1] Operación de registro sobre el servidor
 */
func registro() {
	var password string
	//Introducir usuario y contraseña
	fmt.Println("Introduce email: ")
	email := leerStringConsola()

	/*fmt.Println("Introduce contraseña: ")
	password := leerStringConsola()*/
	//NUEVO: Antes de introducir la contraseña, el sistema preguntará
	fmt.Println("¿Generar contraseña aleatoria? S/N")
	respuestaPassword  := leerStringConsola()

	if strings.EqualFold(respuestaPassword, "s"){
		password = generarContrasenyaAleatoria()
		fmt.Println("El password generado aleatoriamente es ",password)
	}else {
		fmt.Println("Introduce contraseña: ")
		password = leerStringConsola()
	}

	//Generamos el hash del password a partir del password para enviarla al servidor
	//ya con dicho hash
	claveCliente := sha512.Sum512([]byte(password))
	passwordHash := base64.StdEncoding.EncodeToString(claveCliente[0:32]) // una mitad para cifrar datos (256 bits)

	//Generamos los parámetros a enviar al servidor
	parametros := url.Values{}
	parametros.Set("opcion", "1")
	//Pasamos el parámetro a la estructura Usuario
	usuario := Usuario{Email: email, Password: passwordHash}
	parametros.Set("usuario", codificarStructToJSONBase64(usuario))

	//Pasar parámetros al servidor
	cadenaJSON := comunicarServidor(parametros)

	var respuesta Resp
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &respuesta)
	checkError(error)

	//Mostramos lo que nos devuelve como respuesta el servidor
	fmt.Println(respuesta.Msg)
}
/**
*Comprobar pin enviado al usuario por correo
**/
func comprobarPin(){

	//Si no hay usuario actual, no se hace nada
	//if usuarioActual != (Usuario{}) {

		//introducir el pin enviado por correo
		fmt.Println("Introduce pin enviado por correo: ")
		pin := leerStringConsola()

		parametros := url.Values{}
		parametros.Set("opcion", "12")
    parametros.Set("pin", pin)
		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

		//pasar el token
		//parametros.Set("token", codificarStructToJSONBase64(token))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)

		//El servidor devuelve un map de entradas y el CLIENTE
		//lo debe recorrer para mostrarlo por pantalla
		var respuesta Resp

		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)
		fmt.Println(respuesta.Msg)

		if(pinRecibido == pin){
			//fmt.Println(respuesta.Msg)
			token.Dato2=respuesta.Dato.Dato2

		}else{
			fmt.Println(respuesta.Msg)
		 fmt.Println("Debes introducri pin correcto")
		}
//}
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

	//Generamos clave maestra a partir del password que servirá para cifrar/descifrar
	claveCliente := sha512.Sum512([]byte(password))
	passwordHash := base64.StdEncoding.EncodeToString(claveCliente[0:32]) // una mitad para cifrar datos (256 bits)
	claveMaestra = claveCliente[32:64] // una mitad para cifrar datos (256 bits)

	//Generamos los parámetros a enviar al servidor
	parametros := url.Values{}
	parametros.Set("opcion", "2")
	//Pasamos el parámetro a la estructura Usuario
	usuario := Usuario{Email: email, Password: passwordHash}
	parametros.Set("usuario", codificarStructToJSONBase64(usuario))

	//Pasar parámetros al servidor
	cadenaJSON := comunicarServidor(parametros)

	var respuesta RespLogin
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &respuesta)
	checkError(error)
	//var t FactorDoble
	//Manejamos la respuesta de login del servidor
	fmt.Println(respuesta.Ok)
	if respuesta.Ok {
    usuarioActual = usuario
		pinRecibido = respuesta.Pin
		fmt.Println("el pin recibido por correo es "+pinRecibido)
		comprobarPin()

	}else{
		fmt.Println(respuesta.Msg)
	}

	//Mostramos sí o sí lo que nos devuelve como respuesta el servidor
	//fmt.Println(respuesta.Msg)
  // Mostramos el token
	//fmt.Println(respuesta.Dato.Dato2)

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
		fmt.Println(respuesta.Msg)
	}
}

func listarEntradas(){
	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {
		parametros := url.Values{}
		parametros.Set("opcion", "5")

		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

		//pasar el token
		parametros.Set("token", codificarStructToJSONBase64(token))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)
		//El servidor devuelve un map de entradas y el CLIENTE
		//lo debe recorrer para mostrarlo por pantalla
		var respuesta RespEntrada
		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)
		//Recorrer y mostrarfmt.Println(respuesta.Msg)

		for i, m := range respuesta.Entradas {
			fmt.Println(" ----- Entrada ", i, " ----- ")
			fmt.Println("Login: ", m.Login)
			fmt.Println("Password (base64 + AES-CTR): ", m.Password, " - Claro: ", descifrarContrasenyaEntrada(m.Password))
			fmt.Println("Web: ", m.Web)
			fmt.Println("Descripción: ", m.Descripcion)
			fmt.Println(" ----- FIN Entrada ", i, " ----- ")
			fmt.Println()
	  }

	}
}

func crearEntrada(){
	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {
		//Introducir datos de una entrada
		fmt.Println("Introduce login del servicio: ")
		login := leerStringConsola()

		fmt.Println("Introduce contraseña del servicio: ")
		password := leerStringConsola()

		//Encriptamos la contraseña con AES-CRT y utilizando la clave maestra calculada
		//a partir del password del usuario
		passwordCifrado := cifrarContrasenyaEntrada(password)

		fmt.Println("Introduce la web del servicio: ")
		web := leerStringConsola()

		fmt.Println("Introduce una descripción: ")
		descripcion := leerStringConsola()

		parametros := url.Values{}
		parametros.Set("opcion", "7")

		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

		//Pasamos el parámetro a la estructura Entrada
		entrada := Entrada{Login: login, Password: passwordCifrado, Web: web, Descripcion: descripcion}
		parametros.Set("entrada", codificarStructToJSONBase64(entrada))

		//pasar el token
		parametros.Set("token", codificarStructToJSONBase64(token))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)

		var respuesta Resp
		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)
		fmt.Println(respuesta.Msg)
	}
}

func editarEntrada(){
   listarEntradas()
	 var op = ""
	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {

		fmt.Println("Introduce id de la entrada: ")
		id := leerStringConsola()
		i, error := strconv.Atoi(id)
    checkError(error)
		//var respuesta Entrada

		aux  := obtenerEntradasPorId(i)
			if aux!=(Entrada{}) {
				fmt.Println("Login: ",aux.Login)
				fmt.Println("Password: ",descifrarContrasenyaEntrada(aux.Password))
				fmt.Println("Web: ",aux.Web)
				fmt.Println("Descripcion: ",aux.Descripcion)
for{
				fmt.Println("Elige opcion que quieres modificar: ")
				fmt.Println("[1] Login")
				fmt.Println("[2] Password")
				fmt.Println("[3] Web")
				fmt.Println("[4] Descripcion")
				fmt.Println("[q] Para salir")
				op = leerStringConsola()
				println("opcion elegida ",op)

				if op == "1" || op == "2" || op == "3" || op == "4" {

				fmt.Println("Inserta dato: ")
				dato := leerStringConsola()

				switch op {
				case "1":
            aux.Login=dato
					break
				case "2":
						datoCifrado:= cifrarContrasenyaEntrada(dato)
            aux.Password=datoCifrado
					break
				case "3":
	          aux.Web=dato
						break
					case "4":
	          aux.Descripcion=dato
						break

				}
        //respuesta.Entradas[i]=aux
				parametros2 := url.Values{}
				parametros2.Set("id",id)
				parametros2.Set("opcion", "8")

				//Pasamos el parámetro a la estructura Usuario
				parametros2.Set("usuario", codificarStructToJSONBase64(usuarioActual))

				//Pasamos el parámetro a la estructura Entrada

				parametros2.Set("entrada", codificarStructToJSONBase64(aux))

				//pasar el token
				parametros2.Set("token", codificarStructToJSONBase64(token))

				//Pasar parámetros al servidor
				cadenaJSON := comunicarServidor(parametros2)

				var respuesta Resp
				//Des-serializamos el json a la estructura creada
				error := json.Unmarshal(cadenaJSON, &respuesta)
				checkError(error)
				fmt.Println(respuesta.Msg)
				break
			}else if op== "q"{
				break
			}
		}

			}else{
        fmt.Println("No existe Entrada con id ",i)

			}
	}

}

func borrarEntrada(){
	//llamada al metodo listar entrada para demostrar las entradas disponibles
	listarEntradas()
	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {

		fmt.Println("Introduce id de la entrada: ")
		id := leerStringConsola()
    i, error := strconv.Atoi(id)
    checkError(error)

		aux  := obtenerEntradasPorId(i)
			if aux!=(Entrada{}) {
				parametros2 := url.Values{}
				parametros2.Set("id",id)
				parametros2.Set("opcion", "9")

				//Pasamos el parámetro a la estructura Usuario
				parametros2.Set("usuario", codificarStructToJSONBase64(usuarioActual))

				//pasar el token
				parametros2.Set("token", codificarStructToJSONBase64(token))

				//Pasar parámetros al servidor
				cadenaJSON := comunicarServidor(parametros2)

				var respuesta Resp
				//Des-serializamos el json a la estructura creada
				error := json.Unmarshal(cadenaJSON, &respuesta)
				checkError(error)
				fmt.Println(respuesta.Msg)

			}else{
        fmt.Println("No existe Entrada con id ",i)

			}
}
}

func obtenerEntradasPorId(i int) Entrada{

	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {
		parametros := url.Values{}
		parametros.Set("opcion", "5")

		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

		//pasar el token
		parametros.Set("token", codificarStructToJSONBase64(token))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)
		//El SERVIDOR DEBE DEVOLVER UN MAP DE ENTRADAS Y EL CLIENTE RECORRERLO Y MOSTRARLO POR PANTALLA
    var respuesta RespEntrada
		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)

		aux, ok := respuesta.Entradas[i]
			if ok {
				return aux

			}
	}
	return Entrada{}
}

////metodo modificar ///////////////////////////////////////////////////////////
/**
* funcion para modificar contraseña de usuario
*/
func modificarContraseña(){

	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {
    var password string

		//se pregunta si se genera la contraseña de forma aleatoria

		fmt.Println("¿Generar contraseña aleatoria? S/N")
		respuestaPassword  := leerStringConsola()

		if strings.EqualFold(respuestaPassword, "s"){
			password = generarContrasenyaAleatoria()
			fmt.Println("El password generado aleatoriamente es ",password)
		}else {
			fmt.Println("Introduce contraseña: ")
			password = leerStringConsola()
		}

			parametros := url.Values{}
			parametros.Set("opcion", "5")

			//Pasamos el parámetro a la estructura Usuario
			parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))

			//pasar el token
			parametros.Set("token", codificarStructToJSONBase64(token))

			//Pasar parámetros al servidor
			cadenaJSON := comunicarServidor(parametros)
			//El servidor devuelve un map de entradas del usuario que inicia sesion
			var respuesta RespEntrada
			//Des-serializamos el json a la estructura creada
			error := json.Unmarshal(cadenaJSON, &respuesta)
			checkError(error)
			//Recorrer y mostrarfmt.Println(respuesta.Msg)

			for i, m := range respuesta.Entradas {
				 entradas[i] = Entrada{m.Login, descifrarContrasenyaEntrada(m.Password),m.Web, m.Descripcion}
				 fmt.Println(entradas[i])
			}
			//Generamos el hash del password a partir del password para enviarla al servidor
			//ya con dicho hash
			  claveCliente := sha512.Sum512([]byte(password))
			  passwordHash := base64.StdEncoding.EncodeToString(claveCliente[0:32]) // una mitad para cifrar datos (256 bits)
		    claveMaestra = claveCliente[32:64] // una mitad para cifrar datos (256 bits)
	    //fmt.Println(passwordHash, claveMaestra)

			for i, m := range entradas {
				 entradas[i] = Entrada{m.Login, cifrarContrasenyaEntrada(m.Password),m.Web, m.Descripcion}
			}
			//Generamos los parámetros a enviar al servidor
			parametros2 := url.Values{}
			parametros2.Set("opcion", "10")
			//Pasamos el parámetro a la estructura Usuario
			usuario := UsuarioMod{Email: usuarioActual.Email, Password: passwordHash, Entradas: entradas}
			parametros2.Set("usuario", codificarStructToJSONBase64(usuario))
			//pasar el token
			parametros2.Set("token", codificarStructToJSONBase64(token))
			//Pasar parámetros al servidor
			cadenaJSON2 := comunicarServidor(parametros2)

			var respuesta2 Resp
			//Des-serializamos el json a la estructura creada
			error2 := json.Unmarshal(cadenaJSON2, &respuesta2)
			checkError(error2)
			fmt.Println(respuesta2.Msg)
	}

}
func darBajaUsuario(){

	//Si no hay usuario actual, no se hace nada
	if usuarioActual != (Usuario{}) {
		parametros := url.Values{}
		parametros.Set("opcion", "11")

		//Pasamos el parámetro a la estructura Usuario
		parametros.Set("usuario", codificarStructToJSONBase64(usuarioActual))
		//pasar el token
		parametros.Set("token", codificarStructToJSONBase64(token))

		//Pasar parámetros al servidor
		cadenaJSON := comunicarServidor(parametros)
		var respuesta Resp
		//Des-serializamos el json a la estructura creada
		error := json.Unmarshal(cadenaJSON, &respuesta)
		checkError(error)
		fmt.Println(respuesta.Msg)
		//os.Exit(1) //finalizamos el programa

	}
}

////////////////////////////////////////////////////////////////////////////////


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
		fmt.Println("[5] Modificar contraseña usuario")
		fmt.Println("[6] Dar baja usuario")
		fmt.Println("[q] Cerrar sesión")
		opcionElegida := leerStringConsola()

		switch opcionElegida {
		case "1":
			fmt.Println("Se ha elegido listar entradas")
			listarEntradas()
			break
		case "2":
			fmt.Println("Se ha elegido crear entrada")
			crearEntrada()
			break
		case "3":
			fmt.Println("Se ha elegido editar entrada")
			editarEntrada()
			break
		case "4":
			fmt.Println("Se ha elegido borrar entrada")
			borrarEntrada()
			break
		case "5":
			fmt.Println("Se ha elegido modificar contraseña")
			modificarContraseña()
			break
		case "6":
			fmt.Println("Se ha elegido dar baja usuario")
			darBajaUsuario()
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
	return strings.TrimSpace(lectura) // quitamos los espacios
}

/**
* Método para comunicar con el servidor pasándole una serie de parámetros (url.Values)
* Devuelve la respuesta del body codificado en bytes, para que sea el propio
* método invocador el que parsee como desee la respuesta
*/
func comunicarServidor(parametros url.Values)([]byte){

	transporte := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	cliente := &http.Client{Transport: transporte}
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

func cifrarContrasenyaEntrada(pass string)(string){
	return base64.StdEncoding.EncodeToString(encrypt([]byte(pass), claveMaestra))
}

func descifrarContrasenyaEntrada(pass string)(string){
	decode, error := base64.StdEncoding.DecodeString(pass)
	checkError(error)
	cadena := string(decrypt(decode, claveMaestra))
	return cadena
}

// función para cifrar (con AES en este caso), adjunta el IV al principio
func encrypt(data, key []byte) (out []byte) {
	out = make([]byte, len(data)+16)    // reservamos espacio para el IV al principio
	rand.Read(out[:16])                 // generamos el IV
	blk, err := aes.NewCipher(key)      // cifrador en bloque (AES), usa key
	checkError(err)                            // comprobamos el error
	ctr := cipher.NewCTR(blk, out[:16]) // cifrador en flujo: modo CTR, usa IV
	ctr.XORKeyStream(out[16:], data)    // ciframos los datos
	return
}

// función para descifrar (con AES en este caso)
func decrypt(data, key []byte) (out []byte) {
	out = make([]byte, len(data)-16)     // la salida no va a tener el IV
	blk, err := aes.NewCipher(key)       // cifrador en bloque (AES), usa key
	checkError(err)                             // comprobamos el error
	ctr := cipher.NewCTR(blk, data[:16]) // cifrador en flujo: modo CTR, usa IV
	ctr.XORKeyStream(out, data[16:])     // desciframos (doble cifrado) los datos
	return
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
