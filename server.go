package main

import (
	"context" //tenemos dos context, el por defecto y el de sesiones, renombramos éste
	"encoding/json"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"io/ioutil"
	"strconv"
	"crypto/rand"
	"io"
	"golang.org/x/crypto/scrypt" //para instalar: go get "golang.org/x/crypto/scrypt"
															//es un subrepositorio, ruta completa
)

// respuesta por defecto del servidor
type Resp struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
}

// respuesta del servidor con peticiones sobre entradas
type RespEntrada struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	Entradas map[int]Entrada
}

type Usuario struct{
	Email string
	Password string
	Salt string
	Entradas map[int] Entrada
}

type Entrada struct {
    Login string
    Password string
    Web string
    Descripcion string
}

type Sesion struct {
		Email string
		TiempoLimite time.Time
}

//Declaramos y/o inicializamos variables globales
var rutaBBDD = "bbdd.json"
var bbdd *os.File
var usuarios = make(map[string]Usuario)
var entradas = make(map[int]Entrada)
var sesiones = make(map[string]Sesion)

// función para escribir una respuesta del servidor al cliente
func comunicarCliente(w http.ResponseWriter, estructura interface{}) {
	w.Write([]byte(codificarStructToJSONBase64(estructura))) // escribimos el JSON resultante
}

func main() {
	// suscripción SIGINT
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(handler))
	//El clearHandler es por las sesiones, para no provocar:
	//"you need to wrap your handlers with context.ClearHandler as or else you will leak memory!"
	srv := &http.Server{Addr: ":10443", Handler: mux}


	go func() {
		if err := srv.ListenAndServeTLS("cert.pem", "key.pem"); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	//Abrimos la conexión con la base de datos (fichero)
	//Para poder obtener la información
	bbdd, _ = os.Open(rutaBBDD)
	defer bbdd.Close()
	cargarDatos()

	<-stopChan // espera señal SIGINT

	//Cuando hemos pulsado ctrl + c (cerrar servidor)
	//entonces volcamos los datos en la base de datos (fichero bbdd)

	//Parseamos a JSON
	cadenaJSON, err := json.Marshal(usuarios)
	checkError(err)

	//Escribimos en el fichero
	errorEscritura := ioutil.WriteFile(rutaBBDD, cadenaJSON, 0666)
	checkError(errorEscritura)
	log.Println("Volcando datos recopilados en la BBDD...")

	//Mostrar apagado de servidor y apagar el servidor realmente
	log.Println("Apagando servidor ...")

	//Apagar servidor de forma segura
	ctx, fnc := context.WithTimeout(context.Background(), 5*time.Second)
	fnc()
	srv.Shutdown(ctx)
	log.Println("Servidor detenido correctamente")
}

/**
* Maneja las opciones que indica el cliente
*/
func handler(w http.ResponseWriter, request *http.Request) {
	request.ParseForm() // es necesario parsear el formulario
	w.Header().Set("Content-Type", "text/plain") // cabecera estándar
	//Obtenemos la opción/operación solicitada por el cliente
	opcion := request.Form.Get("opcion")

	//Si se intenta desde el cliente insertar una opción que necesita estar logueado y NO está logueado
	// en el servidor, el servidor no le dejará y devolverá error "Operación no permitida..."
	if !tienePermisosOpciones(w, request, opcion){
		opcion = "0"
	}

	switch opcion { // comprobamos comando desde el cliente
		case "0": //caso básico para no permitir
			resp := Resp{Ok: false, Msg: "Operación no permitida. No ha iniciado sesión."} // formateamos respuesta
			comunicarCliente(w, resp)
			break
		case "1": // registro
			registro(w, request)
			break
		case "2": // login
			login(w, request)
			break
		case "3": //Es logueado
			esLogueado(w, request)
			break
		case "4": //Logout
			logout(w, request)
			break
		case "5": //Listar todas las entradas del usuario X
			listarEntradas(w, request)
			break
		case "6": //Listar una entrada por id de entrada
		  obtenerEntradasPorId(w, request)
			break
		case "7": //Crear entrada
			crearEntrada(w, request)
			break
		case "8": //Editar entrada
		  modificarEntrada(w, request)
			break
		case "9": //Borrar entrada
		  borrarEntrada(w, request)
			break
		default:
			resp := Resp{Ok: false, Msg: "Comando inválido"}    // formateamos respuesta
			comunicarCliente(w, resp)
	}
}

/**
* Función que comprueba si existe el usuario en la BBDD
* Recibe como parámetro el email (clave primaria) del usuario
*/
func existeUsuario(email string) (ok bool) {
	if _, ok = usuarios[email]; ok {
	}
	return
}

/**
* Función que comprueba si existe el usuario está logueado
* Recibe como parámetro el email (clave primaria) del usuario
* A diferencia de "esLogueado" esta función es interna del servidor
* por lo que no necesita recibir parámetros de request y response
*/
func esEmailLogueado(email string) (ok bool){
	ok = false
	fechaHoraActual := time.Now()
	if email == sesiones[email].Email &&
			 fechaHoraActual.Before(sesiones[email].TiempoLimite) {
			 ok = true
	}
	return
}

/**
* Comprueba si el usuario tiene permisos para utilizar las opciones propias de un usuario logueado
*/
func tienePermisosOpciones(w http.ResponseWriter, request *http.Request, opcion string) (ok bool) {
	ok = true
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	if !esEmailLogueado(usuario.Email) && opcion != "1" && opcion != "2" && opcion != "3"{
		ok = false
	}
	return
}

/**
* Registramos al usuario, para ello hay que asegurarse que no exista
* ya (tener mismo login)
*/
func registro(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	r := Resp{}
	//Comprobamos que el usuario existe en la base de datos
	if existeUsuario(usuario.Email){
		r = Resp{Ok: false, Msg: "El usuario ya existe, vuelve a intentarlo con otros datos."}    // formateamos respuesta
	}else{
		//Si no existe, procedemos a crearlo
		//Inicializamos las entradas como vacías (lógicamente no tiene ninguna)
		usuario.Entradas = make(map[int]Entrada)

		//Generamos el salt aleatorio para la contraseña
		salt := make([]byte, 32)
    _, error2 := io.ReadFull(rand.Reader, salt)
    checkError(error2)
		//Recibimos del cliente base64(SHA256(passwordIntroducidaConsola))
		//Se realiza ahora -> scrypt(decodebase64(SHA256(passwordIntroducidaConsola)), salt)
		pass, error3 := base64.StdEncoding.DecodeString(usuario.Password)
		checkError(error3)
    hash, error4 := scrypt.Key(pass, salt, 16384, 8, 1, 32)
		checkError(error4)
		usuario.Password = base64.StdEncoding.EncodeToString(hash)
		usuario.Salt = base64.StdEncoding.EncodeToString(salt)
		//Lo agregamos al mapa global de usuarios
		usuarios[usuario.Email] = usuario
		r = Resp{Ok: true, Msg: "Registrado con éxito. Inicia sesión para empezar."}    // formateamos respuesta
	}
	comunicarCliente(w, r)
}

func login(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	r := Resp{}
	//Comprobamos que el usuario existe en la base de datos
	if existeUsuario(usuario.Email){
		//Ahora comprobamos si Email y Contraseña enviada
		//desde cliente coincide con lo que tenemos de dicho usuario en la bbdd
		if usuarios[usuario.Email].Email == usuario.Email {
			//Generamos el hash con el password pasado como parámetro
			//y se compara con el que hay insertado en la bbdd (utiliza la salt almacenada en la bbdd)
			pass, error2 := base64.StdEncoding.DecodeString(usuario.Password)
			checkError(error2)
			salt, error3 := base64.StdEncoding.DecodeString(usuarios[usuario.Email].Salt)
			checkError(error3)
			hash, error4 := scrypt.Key(pass, salt, 16384, 8, 1, 32)
			checkError(error4)
			passIntroducidoCliente := base64.StdEncoding.EncodeToString(hash)
			if usuarios[usuario.Email].Password == passIntroducidoCliente {
				//Se crea la sesión con tiempo actual + 90 segundos de tiempo límite
				sesion := Sesion{Email: usuario.Email, TiempoLimite: time.Now().Add(time.Hour * time.Duration(0) +
                                 time.Minute * time.Duration(1) +
                                 time.Second * time.Duration(30))}
				sesiones[usuario.Email] = sesion
				r = Resp{Ok: true, Msg: "El usuario se ha logueado correctamente."}    // formateamos respuesta
			}else{
				r = Resp{Ok: false, Msg: "La contraseña no es correcta. Vuelva a intentarlo."}    // formateamos respuesta
			}
	  }else{
			r = Resp{Ok: false, Msg: "No coinciden los parámetros del usuario. Vuelve a intentarlo."}    // formateamos respuesta
		}
	}else{
		r = Resp{Ok: false, Msg: "El usuario no existe, regístrate y vuelve a intentarlo."}    // formateamos respuesta
	}
	comunicarCliente(w, r)
}

/**
* Comprueba si el usuario está con la sesión iniciada y devuelve una respuesta al cliente
*/
func esLogueado(w http.ResponseWriter, request *http.Request) {
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	r := Resp{}

	//Comprobamos tanto si el usuario existe en el map de sesiones como si su datetime
	//no ha pasado
	if esEmailLogueado(usuario.Email) {
		 r = Resp{Ok: true, Msg: "El usuario está logueado correctamente."} // formateamos respuesta
	} else{
		 r = Resp{Ok: false, Msg: "El usuario no está logueado."} // formateamos respuesta
	}
	comunicarCliente(w, r)
}

func logout(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	r := Resp{}
	if esEmailLogueado(usuario.Email) {
		//Cerrar sesión
		//Borramos del mapa de sesiones
		delete(sesiones, usuario.Email)
		r = Resp{Ok: true, Msg: "Sesión cerrada con éxito."}    // formateamos respuesta
	} else{
		r = Resp{Ok: false, Msg: "La sesión ya está cerrada."}
	}

	comunicarCliente(w, r)
}

func crearEntrada(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	cadenaJSONEntrada := decodificarJSONBase64ToJSON(request.Form.Get("entrada"))

	var usuario Usuario
	var entrada Entrada
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONEntrada, &entrada)
	checkError(error2)
	r := Resp{}
	//Si está logueado el usuario en el sistema, entonces podemos crear la entrada
	//Si no, error ya que se le ha acabado la sesión
	if esEmailLogueado(usuario.Email) {
		//Las entradas empezarán en el 1
		usuarios[usuario.Email].Entradas[len(usuarios[usuario.Email].Entradas)+1] = entrada
		r = Resp{Ok: true, Msg: "Entrada creada con éxito."}
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
	}
	//anyadirEntrada()
	comunicarCliente(w, r)
}

func listarEntradas(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))

	var usuario Usuario

	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)

	r:= RespEntrada{}
	if esEmailLogueado(usuario.Email) {
		r = RespEntrada{Ok: true, Msg: "Devolviendo entradas.", Entradas: usuarios[usuario.Email].Entradas}
	} else {
		r = RespEntrada{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión.", Entradas: make(map[int]Entrada)}
	}
	comunicarCliente(w, r)
}

/*
* Funcion para modificar una entrada
*/
func modificarEntrada(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	cadenaJSONEntrada := decodificarJSONBase64ToJSON(request.Form.Get("entrada"))
	opcion := request.Form.Get("id")

	var usuario Usuario
	var entrada Entrada
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONEntrada, &entrada)
	checkError(error2)
	r := Resp{}
	//Si está logueado el usuario en el sistema, entonces podemos crear la entrada
	//Si no, error ya que se le ha acabado la sesión
	if esEmailLogueado(usuario.Email) {
		i, error := strconv.Atoi(opcion)
		checkError(error)
		//se pasa el id de la entrada al que se pretende modificar
		usuarios[usuario.Email].Entradas[i]= entrada
		r = Resp{Ok: true, Msg: "Entrada Modificada con éxito."}
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
	}
	comunicarCliente(w, r)
}

/*
*Funcion para borrar una entrada de un usuario
*/
func borrarEntrada(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	opcion := request.Form.Get("id")

	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)

	r := Resp{}
	if esEmailLogueado(usuario.Email) {
		i, error := strconv.Atoi(opcion)
		checkError(error)
		//se pasa el id de la entrada al que se pretende borrar
		delete(usuarios[usuario.Email].Entradas, i)
		r = Resp{Ok: true, Msg: "Entrada Borrada con éxito."}
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
	}

	comunicarCliente(w, r)

}

func obtenerEntradasPorId(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario

	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)

	r:= RespEntrada{}
	if esEmailLogueado(usuario.Email) {
		   r = RespEntrada{Ok: true, Msg: "Devolviendo entrada.",Entradas: usuarios[usuario.Email].Entradas}
	} else {
		   r = RespEntrada{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión.", Entradas: make(map[int]Entrada)}
	}
	comunicarCliente(w, r)


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
* Función que se encarga de obtener todos los datos del fichero
* para trabajar en memoria
*/
func cargarDatos(){
	datosBBDD, error := ioutil.ReadAll(bbdd)
	checkError(error)
	json.Unmarshal(datosBBDD, &usuarios)
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
