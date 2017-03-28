package main

import (
	senyal "context" //tenemos dos context, el por defecto y el de sesiones, renombramos éste
	"encoding/json"
	"encoding/base64"
	"fmt"
	"strconv"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"
	"io/ioutil"
	"github.com/gorilla/context"
	"github.com/gorilla/sessions"
	"github.com/gorilla/securecookie"
)

// respuesta del servidor
type Resp struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	SessionId string
}

type Usuario struct{
	Email string
	Password string
	Entradas map[int] Entrada
}

type Entrada struct {
    Login string
    Password string
    Web string
    Descripcion string
}

//Declaramos y/o inicializamos variables globales
var rutaBBDD = "bbdd.json"
var bbdd *os.File
var usuarios = make(map[string]Usuario)
var entradas = make(map[int]Entrada)
var sesiones *sessions.CookieStore

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
	srv := &http.Server{Addr: ":10443", Handler: context.ClearHandler(mux)}

	//Las opciones de las sesiones
	sesiones = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))
	sesiones.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   10, // 8 hours
		HttpOnly: true,
		Secure: true,
	}

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
	ctx, fnc := senyal.WithTimeout(senyal.Background(), 5*time.Second)
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

	if sesion := obtenerSesionPorId(w, request); sesion != nil {
		//Devolvemos tan solo la cookie del usuario X
		sesion.Save(request, w)
	} else if opcion != "1" && opcion != "2" && opcion != "3" {
		opcion = "0"
	}

	switch opcion { // comprobamos comando desde el cliente
		case "0": //caso básico para no permitir
			resp := Resp{Ok: false, Msg: "Operación no permitida. No ha iniciado sesión."} // formateamos respuesta
			comunicarCliente(w, resp)
		case "1": // registro
			registro(w, request)
			break
		case "2": // login
			login(w, request)
		case "3": //Es logueado
			esLogueado(w, request)
		case "4": //Logout
			logout(w, request)
		case "5": //Listar todas las entradas

		case "6": //Listar una entrada por id

		case "7": //Crear entrada

		case "8": //Editar entrada

		case "9": //Borrar entrada

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
		if usuarios[usuario.Email].Email == usuario.Email &&
			usuarios[usuario.Email].Password == usuario.Password{
			//Se crea la sesión
	    sesion, error2 := sesiones.Get(request, "sesionid")
	    checkError(error2)
	    sesion.Values["email"] = usuario.Email
	    // Guardamos la sesion, antes de que se responda al cliente llamando a "comunicarCliente"
			sesion.Save(request, w)
			r = Resp{Ok: true, Msg: "El usuario se ha logueado correctamente."}    // formateamos respuesta
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
	sesion, err := sesiones.Get(request, "sesionid")
	//La sesión (cookie) tiene como valor el email del usuario
	fmt.Println("EL MAXIMO EDAD ES: "+strconv.Itoa(sesion.Options.MaxAge))
	if err == nil && !sesion.IsNew && sesion.Options.MaxAge>0 && (sesion.Values["email"] == usuario.Email) {
		fmt.Println("Logueadoooo")
		r = Resp{Ok: true, Msg: "El usuario está logueado correctamente."}    // formateamos respuesta
	} else{
		fmt.Println("No logueadooo")
		r = Resp{Ok: false, Msg: "El usuario no está logueado."}    // formateamos respuesta
	}
	comunicarCliente(w, r)
}

func logout(w http.ResponseWriter, request *http.Request){
	r := Resp{}
	if sesion := obtenerSesionPorId(w, request); sesion != nil {
		//Establecemos el valor a vacío y el tiempo de expiración a nulo para borrarlo
		sesion.Values["email"] = ""
		sesion.Options.MaxAge = -10
		sesion.Save(request, w)
		w.Header().Set("Expires", "Tue, 03 Jul 2001 06:00 GMT")
		w.Header().Set("Last-Modified", "{now} GMT")
		w.Header().Set("Cache-Control", "max-age=0, no-cache, must-revalidate, proxy-revalidate")

		r = Resp{Ok: true, Msg: "Sesión cerrada con éxito."}    // formateamos respuesta
	} else{
		r = Resp{Ok: false, Msg: "No se ha podido cerrar sesión. Ha ocurrido algún error inesperado."}    // formateamos respuesta
	}
	comunicarCliente(w, r)
}
/**
* Comprueba si el usuario está logueado y devuelve la sesión.
* A diferencia de "esLogueado", este método es interno a nivel del server y no se comunicar
* con el cliente
*/
func obtenerSesionPorId(w http.ResponseWriter, request *http.Request)(*sessions.Session){
	var respuesta *sessions.Session
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	sesion, err := sesiones.Get(request, "sesionid")
	//La sesión (cookie) tiene como valor el email del usuario
	if err == nil && !sesion.IsNew && sesion.Options.MaxAge>0 && (sesion.Values["email"] == usuario.Email) {
		respuesta = sesion
	}
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
