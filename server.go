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
  jwt "github.com/dgrijalva/jwt-go"
	X "math/rand"
	"net/smtp"
	"net/mail"
	"strings"
)

/**
* respuesta despues de comprobar si el usuario esta en la base de datos
*/
type RespLogin struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	//Dato Token // el token
	//Pin string
}

/**
* respuesta por defecto del servidor
*/
type Resp struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	Dato Token // el token
	//Pin string
}

/**
* respuesta del servidor con peticiones sobre entradas
*/
type RespEntrada struct {
	Ok  bool   // true -> correcto, false -> error
	Msg string // mensaje adicional
	Entradas map[int]Entrada
}

/**
* Estructura usuario con todos sus componentes
*/
type Usuario struct{
	Email string
	Password string
	//Pin string
	Salt string
	Entradas map[int] Entrada
}

/**
* Estructura entrada con todos sus componentes
*/
type Entrada struct {
    Login string
    Password string
    Web string
    Descripcion string
}

/**
* Estructura entrada con todos sus componentes
*/
type Token struct{
  Dato2 string
}

/**
* Estructura sesion con el tiempo límite a comparar
*/
type Sesion struct {
		Email string
		TiempoLimite time.Time
		Dato Token
}

/**
* Estructura Sesión Pin con el tiempo límite
*/
type SesionPin struct {
		TiempoLimite time.Time
}

//Declaramos y/o inicializamos variables globales
var rutaBBDD = "bbdd.json"
var bbdd *os.File
var usuarios = make(map[string]Usuario)
var entradas = make(map[int]Entrada)
var sesiones = make(map[string]Sesion)
var lettersNumbers = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
var pin string
var sesionPin SesionPin

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

 //create your file with desired read/write permissions
        f, err := os.OpenFile("log.txt", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
        if err != nil {
                checkError(err)
        }

				//defer to close when you're done with it, not because you think it's idiomatic!
        defer f.Close()
        //set output of logs to f
         log.SetOutput(f)
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
		case "10": //Modificar contraseña de usuario
			modificarUsuario(w, request)
		case "11": //Modificar contraseña de usuario
			darBajaUsuario(w, request)
			break
			case "12": //Comprobar si el pin del usuario es correcto
				comprobarPin(w, request)
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
	if !esEmailLogueado(usuario.Email) && opcion != "1" && opcion != "2" && opcion != "3" && opcion != "12" {
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
		log.Println("Usuario "+ usuario.Email + " existe en la base de datos, no se puede registrar dos veces")

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
		fmt.Println("la contraseña es "+ usuario.Password)
		//pin := randLettersNumbers(10)
		//usuario.Pin = pin
		//Lo agregamos al mapa global de usuarios
		usuarios[usuario.Email] = usuario
		r = Resp{Ok: true, Msg: "Registrado con éxito. Inicia sesión para empezar."}    // formateamos respuesta
		//test case
    log.Println("Usuario "+ usuario.Email + " se ha registrado correctamente\n")

	}
	comunicarCliente(w, r)
}
/**
*funcion para devolver caracteres aleatorios
*/
	func randLettersNumbers(n int) string {
	    b := make([]rune, n)
			X.Seed(time.Now().UnixNano())
	    for i := range b {
	        b[i] = lettersNumbers[X.Intn(len(lettersNumbers))]
	    }
	    return string(b)
	}
//*jwt.Token
func crearToken(userEmail string) string{
	mySigningKey := []byte(randLettersNumbers(50))
	//fmt.Println("random:",mySigningKey)
	//mySigningKey := []byte(userEmail)

	// Create the Claims
	claims := &jwt.StandardClaims{
			//NotBefore: int64(time.Now().Unix() - 1000),
			ExpiresAt: int64(time.Now().Unix() + 90),
			Issuer:    userEmail,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	ss, err := token.SignedString(mySigningKey)
	checkError(err)

	return ss
}
func login(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	var usuario Usuario
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSON, &usuario)
	checkError(error)
	r := RespLogin{}
	//Comprobamos que el usuario existe en la base de datos
	if existeUsuario(usuario.Email){
		//Ahora comprobamos si Email y Contraseña enviada
		//desde cliente coincide con lo que tenemos de dicho usuario en la bbdd
		if usuarios[usuario.Email].Email == usuario.Email{
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
        //generar el pin para mandarselo al usuario por correo
				pin = randLettersNumbers(10)
				senMail(usuario.Email,pin)
				//crear una sesion del pin mandado al usuario por correo
				sesionPin = SesionPin{TiempoLimite: time.Now().Add(time.Hour * time.Duration(0) +
											time.Minute * time.Duration(5) +
											time.Second * time.Duration(00)) }

				r = RespLogin{Ok: true, Msg: "El login y la contraseña son correctos, en breve recibirás un pin para introducir."  }    // formateamos respuesta
				log.Println("Usuario "+ usuario.Email + " se ha logeado correctamente")
			}else{
				r = RespLogin{Ok: false, Msg: "La contraseña no es correcta. Vuelva a intentarlo."}    // formateamos respuesta
				log.Println("Usuario "+ usuario.Email + " ha introducido contraseña incorrecta")
			}
	  }else{
			r = RespLogin{Ok: false, Msg: "No coinciden los parámetros del usuario. Vuelve a intentarlo."}    // formateamos respuesta
			log.Println("Usuario "+ usuario.Email + " se ha puesto datos erroneos")
		}
	}else{
		r = RespLogin{Ok: false, Msg: "El usuario no existe, regístrate y vuelve a intentarlo."}    // formateamos respuesta
		log.Println("Usuario "+ usuario.Email + " no existe en la base de datos")
	}
	comunicarCliente(w, r)
}

func comprobarPin(w http.ResponseWriter, request *http.Request){
//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
cadenaJSON := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
//cadenaPin := decodificarJSONBase64ToJSON(request.Form.Get("pin"))
cadenaPin := request.Form.Get("pin")
var usuario Usuario

//Des-serializamos el json a la estructura creada
error := json.Unmarshal(cadenaJSON, &usuario)
checkError(error)
r := Resp{}

//Comprobamos que el usuario existe en la base de datos
if existeUsuario(usuario.Email){
	//Ahora comprobamos si Email y Contraseña enviada
	//desde cliente coincide con lo que tenemos de dicho usuario en la bbdd
	//se comprueba si la sesion del pin, todavia esta activada
	fechaHoraActual := time.Now()
	if fechaHoraActual.Before(sesionPin.TiempoLimite) {
		if usuarios[usuario.Email].Email == usuario.Email &&  pin == string(cadenaPin) {
			//se genera el token
			var token Token
			token.Dato2 = crearToken(usuario.Email)
			//Se crea la sesión con tiempo actual + 90 segundos de tiempo límite
	           sesion := Sesion{Email: usuario.Email, TiempoLimite: time.Now().Add(time.Hour * time.Duration(0) +
													 time.Minute * time.Duration(1) +
													 time.Second * time.Duration(30)), Dato : token }
	           sesiones[usuario.Email] = sesion
			r = Resp{Ok: true, Msg: "El pin introducido es correcto.", Dato: token }    // formateamos respuesta
			log.Println("Usuario "+ usuario.Email + " ha puesto pin correcto")
		}else{
			r = Resp{Ok: false, Msg: "El pin no es correcto. Vuelve a intentarlo.", Dato: Token{}}    // formateamos respuesta
			log.Println("Usuario "+ usuario.Email + " ha puesto pin incorrecto")
		}
	}else{
		r = Resp{Ok: false, Msg: "Se ha caducado el tiempo de sesion del pin.", Dato: Token{}}    // formateamos respuesta
		log.Println("Usuario "+ usuario.Email + " ha tardado en poner el pin")

	}
 }else{
		r = Resp{Ok: false, Msg: "El usuario no existe, regístrate y vuelve a intentarlo.", Dato: Token{}}    // formateamos respuesta
		log.Println("Usuario "+ usuario.Email + " no existe en la base de datos")
}
comunicarCliente(w, r)

}
/**
*
*/
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
		log.Println("Usuario "+ usuario.Email + " ha cerrado sesion")
	} else{
		r = Resp{Ok: false, Msg: "La sesión ya está cerrada."}
		log.Println("Usuario "+ usuario.Email + " intenta cerrar sesion, y esta cerrada antes")
	}

	comunicarCliente(w, r)
}

func crearEntrada(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	cadenaJSONEntrada := decodificarJSONBase64ToJSON(request.Form.Get("entrada"))
  cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))
	var usuario Usuario
	var entrada Entrada
	var token Token
	cont := 1
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONEntrada, &entrada)
	checkError(error2)
	error3 := json.Unmarshal(cadenaJSONtoken, &token)
	checkError(error3)
	r := Resp{}
	//Si está logueado el usuario en el sistema, entonces podemos crear la entrada
	//Si no, error ya que se le ha acabado la sesión
	if esEmailLogueado(usuario.Email) &&  sesiones[usuario.Email].Dato.Dato2==token.Dato2{
		for {
			if _,ok := usuarios[usuario.Email].Entradas[cont]; ok{
				 cont++
			}else{
				//fmt.Println("ha entrado a crear una entrada nueva")
			usuarios[usuario.Email].Entradas[cont] = entrada
			//len(usuarios[usuario.Email].Entradas)+1
			break
		}
		}


		r = Resp{Ok: true, Msg: "Entrada creada con éxito."}
		log.Println("Usuario "+ usuario.Email + " ha creado nueva entrada")
		//llamada a la funcion para reiniciar la sesion
    reiniciarSesion(usuario)
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
		log.Println("Usuario "+ usuario.Email + " inetenta crear entrada con sesion cerrada")
	}
	//anyadirEntrada()
	comunicarCliente(w, r)
}

func listarEntradas(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
  cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))
	var usuario Usuario
  var token Token
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONtoken, &token)
	checkError(error2)
	r:= RespEntrada{}

	if esEmailLogueado(usuario.Email) &&  sesiones[usuario.Email].Dato.Dato2==token.Dato2{
		r = RespEntrada{Ok: true, Msg: "Devolviendo entradas.", Entradas: usuarios[usuario.Email].Entradas}
		log.Println("Usuario "+ usuario.Email + " solicita sus entradas")
		//llamada a la funcion para reiniciar la sesion
    reiniciarSesion(usuario)
	} else {
		r = RespEntrada{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión.", Entradas: make(map[int]Entrada)}
		log.Println("Usuario "+ usuario.Email + " solicita listar entradas, cuando ha perdido la sesion")
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
	cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))

	var usuario Usuario
	var entrada Entrada
	var token Token
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONEntrada, &entrada)
	checkError(error2)
	error3 := json.Unmarshal(cadenaJSONtoken, &token)
	checkError(error3)
	r := Resp{}
	//Si está logueado el usuario en el sistema, entonces podemos crear la entrada
	//Si no, error ya que se le ha acabado la sesión
	if esEmailLogueado(usuario.Email) && sesiones[usuario.Email].Dato.Dato2==token.Dato2{
		i, error := strconv.Atoi(opcion)
		checkError(error)
		//se pasa el id de la entrada al que se pretende modificar
		usuarios[usuario.Email].Entradas[i]= entrada
		r = Resp{Ok: true, Msg: "Entrada Modificada con éxito."}
		log.Println("Usuario "+ usuario.Email + " solicita modificar entrada con exito")
		//llamada a la funcion para reiniciar la sesion
    reiniciarSesion(usuario)
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
		log.Println("Usuario "+ usuario.Email + " solicita modificar entrada, cuando ha perdido la sesion")
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
  cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))
	var usuario Usuario
	var token Token
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONtoken, &token)
	checkError(error2)

	r := Resp{}
	if esEmailLogueado(usuario.Email) && sesiones[usuario.Email].Dato.Dato2==token.Dato2 {
		i, error := strconv.Atoi(opcion)
		checkError(error)
		//se pasa el id de la entrada al que se pretende borrar
		delete(usuarios[usuario.Email].Entradas, i)
		r = Resp{Ok: true, Msg: "Entrada Borrada con éxito."}
		log.Println("Usuario "+ usuario.Email + " solicita borrar entrada")
		//llamada a la funcion para reiniciar la sesion
    reiniciarSesion(usuario)
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
		log.Println("Usuario "+ usuario.Email + " solicita borrar entrada, cuando ha perdido la sesion")
	}

	comunicarCliente(w, r)

}

func obtenerEntradasPorId(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))

	var usuario Usuario
  var token Token
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONtoken, &token)
	checkError(error2)
	r:= RespEntrada{}
	if esEmailLogueado(usuario.Email) && sesiones[usuario.Email].Dato.Dato2==token.Dato2{
		   r = RespEntrada{Ok: true, Msg: "Devolviendo entrada.",Entradas: usuarios[usuario.Email].Entradas}
			 log.Println("Usuario "+ usuario.Email + " solicita entrada por identificador")
			 //llamada a la funcion para reiniciar la sesion
	     reiniciarSesion(usuario)
	} else {
		   r = RespEntrada{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión.", Entradas: make(map[int]Entrada)}
			 log.Println("Usuario "+ usuario.Email + " solicita entrada por identificador, cuando ha perdido la sesion")
	}
	comunicarCliente(w, r)
}
/*
* funcion para dar baja a un cliente
*/
func darBajaUsuario(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))
	var usuario Usuario
	var token Token
	//Des-serializamos el json a la estructura creada
	error := json.Unmarshal(cadenaJSONUsuario, &usuario)
	checkError(error)
	error2 := json.Unmarshal(cadenaJSONtoken, &token)
	checkError(error2)

	r := Resp{}
	if esEmailLogueado(usuario.Email) && sesiones[usuario.Email].Dato.Dato2==token.Dato2 {
		delete(usuarios,usuario.Email)
		delete(sesiones, usuario.Email)
		r = Resp{Ok: true, Msg: "Usuario Borrado con éxito."}
		log.Println("Usuario "+ usuario.Email + " solicita dar del baja del sistema")
	} else {
		r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
		log.Println("Usuario "+ usuario.Email + " solicita dar de baja del sistema, cuando ha perdido la sesion")
	}

	comunicarCliente(w, r)

}
/**
*Modificar contraseña de usuario
*/
func modificarUsuario(w http.ResponseWriter, request *http.Request){
	//Viene del cliente codificado en JSON en base64, lo pasamos a JSON simple
	cadenaJSONUsuario := decodificarJSONBase64ToJSON(request.Form.Get("usuario"))
	cadenaJSONtoken := decodificarJSONBase64ToJSON(request.Form.Get("token"))

	var usuario Usuario
  var token Token
  //Des-serializamos el json a la estructura creada
  error := json.Unmarshal(cadenaJSONUsuario, &usuario)
  checkError(error)
  error2 := json.Unmarshal(cadenaJSONtoken, &token)
  checkError(error2)
  r:= Resp{}
  if esEmailLogueado(usuario.Email){
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
		r = Resp{Ok: true, Msg: "Contraseña modificada con éxito."}    // formateamos respuesta
		log.Println("Usuario "+ usuario.Email + "  modificar contraseña con éxito")
		//llamada a la funcion para reiniciar la sesion
    reiniciarSesion(usuario)
  } else {
 			r = Resp{Ok: false, Msg: "Operación no puede completarse, el usuario ha perdido la sesión."}
			log.Println("Usuario "+ usuario.Email + " solicita modificar entrada, cuando ha perdido la sesión")
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
* funcion para reiniciar sesion
**/
func reiniciarSesion(usuario Usuario){

	//Cuando el usuario realiza una operacion, se reinicia la sesion, para mantener la sesion
	sesion := sesiones[usuario.Email]
	sesion.TiempoLimite = time.Now().Add(time.Hour * time.Duration(0) +
								time.Minute * time.Duration(1) +
								time.Second * time.Duration(30))
							sesiones[usuario.Email] = sesion
}
/*
*Funcion para mandar el email
*/

func senMail(email,pin string){

	//Cuenta de GMAIL creada para enviar emails con los pins a los
	//usuarios
	auth := smtp.PlainAuth(
		"",
		"sdsua2017@gmail.com",
		"sds2017ua",
		"smtp.gmail.com",
	)
	from := mail.Address{"Gestor de contraseñas - SDS UA", "sdsua2017@gmail.com"}
	to := mail.Address{email, email}
	title := "Nuevo pin - Gestor de contraseñas"

	body := "Hola, \n\nBienvenido a Gestor de contraseñas.\n" +
	"Para poder acceder al sistema necesitarás un pin generado automáticamente y de forma segura. \n" +
	"Es el siguiente:\n\n" +  pin + "\nEsperemos que disfrutes de nuestro servicio." +
	"\n\nMuchas gracias.";

	header := make(map[string]string)
	header["From"] = from.String()
	header["To"] = to.String()
	header["Subject"] = encodeRFC2047(title)
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/plain; charset=\"utf-8\""
	header["Content-Transfer-Encoding"] = "base64"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(body))

	error := smtp.SendMail(
		 "smtp.gmail.com:587",
		auth,
		from.Address,
		[]string{to.Address},
		[]byte(message),

	)
	checkError(error)
	log.Println("Usuario "+ email + " se le ha enviado pin para login con exito")
}
func encodeRFC2047(String string) string{

	addr := mail.Address{String, ""}
	return strings.Trim(addr.String(), " <>")
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
