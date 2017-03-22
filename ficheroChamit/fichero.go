package main

import (
    "fmt"
    "os"
    //"bufio"
    //"strings"
    "encoding/json"
    "io/ioutil"

)
type Entrada struct {

    IdUsuario int
    Login string
    Password string
    Web string
    Descripcion string

}


type Usuario struct {
    User string
    Password string
    Entradas map[int] Entrada
}
//password:=dbusuarios["pepe"].entrda["pepe@fb"]
//pepe
//-pepe@fb/mipass/micomentario de fb
//-pepetwit/mipass2/micomentario de twitter
//-pepeinsta/mipass3/mi coemnetario de instagram

var fout *os.File
var slice = make([]Entrada,0)
var mapa =make(map[string]Usuario)

func main() {
	//var fout *os.File // fichero de salida
//  var err error     // receptor de error
var dato1 Entrada
var dato2 Entrada
var dato3 Entrada
var dato4 Entrada
var aux Usuario
aux.Entradas= make(map[int]Entrada)


//var s     Entrada

    dato1.IdUsuario=1
    dato1.Login="vv"
    dato1.Password="xx"
    dato1.Web="www.twitter.com"
    dato1.Descripcion="lo que sea"


    dato2.IdUsuario=2
    dato2.Login="vv"
    dato2.Password="yy"
    dato2.Web="www.twitter.com"
    dato2.Descripcion="lo que sea"

    //usuario 2
    dato3.IdUsuario=3
    dato3.Login="xx"
    dato3.Password="xx"
    dato3.Web="www.twitter.com"
    dato3.Descripcion="lo que sea"


    dato4.IdUsuario=3
    dato4.Login="xx"
    dato4.Password="yy"
    dato4.Web="www.twitter.com"
    dato4.Descripcion="lo que sea"
    //conseguirEntradas(os.Args[1])
  //  user.usuario="vv"
    aux.User=dato1.Login
    aux.Password=dato1.Password
    aux.Entradas[1]=dato1
    aux.User=dato2.Login
    aux.Password=dato2.Password
    aux.Entradas[2]=dato2
    mapa[dato1.Login]=aux

    aux.User=dato3.Login
    aux.Password=dato3.Password
    aux.Entradas[1]=dato3
    aux.User=dato4.Login
    aux.Password=dato4.Password
    aux.Entradas[2]=dato4
    mapa[dato3.Login]=aux


   altaEntrada(os.Args[1])
   conseguirEntradas(os.Args[1])
   obtenerEntradasPorId("vv")
   modificar("vv","PasswordXXXX",1,os.Args[1])
   borrarEntrada("vv",1,os.Args[1])
   anyadirEntradaUsuario("vv",10,os.Args[1])
//demostrarLineasUsuario(slice)
}
func borrarEntrada(usuario string,id int,path string){

  delete(mapa[usuario].Entradas, id)
  altaEntrada(path)
}
func conseguirEntradas(path string){
  //var s Entrada
  fout, err := os.Open(path)
  check(err)
  defer fout.Close()
   data, err := ioutil.ReadAll(fout)
    check(err)
   json.Unmarshal(data, &mapa)

}
func altaEntrada(path string){

  b, err := json.Marshal(mapa)
  check(err)
  er := ioutil.WriteFile(path, []byte(b), 0666)
  check(er)

}
func obtenerEntradasPorId(usuario string){

  for _, m := range mapa[usuario].Entradas {
         fmt.Println( m)
  }
}

func modificar(usuario,dato string,id int,path string){

  //p:=mapa[usuario].Entradas[id]
  m:=mapa[usuario].Entradas[id]
  m.Password=dato
  mapa[usuario].Entradas[id]=m
  altaEntrada(path)
}
func anyadirEntradaUsuario(usuario string,id int,path string){
var datoX Entrada

//  m := mapa[usuario]


  datoX.IdUsuario=1
  datoX.Login="vv"
  datoX.Password="xx"
  datoX.Web="www.twitter.com"
  datoX.Descripcion="lo que sea"
  mapa[usuario].Entradas[id]=datoX
  altaEntrada(path)


}

func listarUsuarios(slice[]string){
  for i := 0; i < len(slice); i++ {
      fmt.Printf("slice[%d] == %s\n",i, slice[i])
      }
  }


func check(e error) {
    if e != nil {
        panic(e)
    }
}
