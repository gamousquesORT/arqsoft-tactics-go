/*
	este ejemplo introduce el uso de channels para sincronizar las goroutines.

	Los channels permiten "pasar" datos entre corruntinas y utilizarlos como forma de coordinación

	Este ejemplo se basa en el código de los ejemplos anteriores pero en lugar de imprimir los ruesultados
	del comando hhtp HEAD a consola, se pasan mediante un canal de nuevo a la gorutina padre (recoradar que el código 
	que lanza las gorutinas es tambien una!)

	al ejecutar presetar atención al orden en que se depliega el workerId o el url y
	a la cantidad de segundos que duró la ejecución (con go test -bench)

	el ejemplo se adapta de https://github.com/quii/learn-go-with-tests/tree/main/concurrency
	y el libro Concurrency in Go por Katherine Cox-Buday
	*/

package main

import (
	"fmt"
	"net/http"
)

// esta función es la que llama a las goroutinas y las sincroniza
// utilizando sync.WaitGroups
func CheckWebsites(urls []string)  {
	
	var workers int = 1;

	// se crea un canal que permite pasar strings https://gobyexample.com/channels
	resultStream := make(chan string)

	// recorremos el slice de urls y lanzamos una gorutina para que verifiquen en forma concurrente los
	// sitios. el resultado se pasa por el canal resultStrean. 
	// NOTAR que cuando este for termine de ejecutar se sigue ejecutando este código en concurrencia con
	// las gorutinas! o sea, el siguiente for .... se ejecuta concurremente sacando los datos del canal
	for _, url := range urls {
		
		go CheckOneWebsite(url, workers, resultStream)
		workers++
	}
	
	//este for se comienza a ejecutar luego de haberse lanzado las gorutimas
	// va sacando del canal a media que las goruntinas agregan
	for i := 0; i < len(urls); i++ { 
		strVal := <- resultStream
		fmt.Printf("%s ",strVal)
	}
	//se cierra el canal porque no se usa mas, todas las gorutinas terminaron y esta 
	//ya desplego todos los datos
	close(resultStream)
}



// esta función hace HEAD de el URL e imprime si responde o no
// notar el último parametro que es un canal de escritura
func CheckOneWebsite(url string,  workerId int, results chan<-string)  {
	
	// el que sigue es el mismo código que en el ejemplo secuencial, solo que con un canal
	// las funcion Sprintf crea un string que se pasa al canal
	response, err := http.Head(url)
	if err != nil {
		results <- fmt.Sprintf("workerid: %d para url %s no existe \n", workerId, url)
	} else { 

		// ok Head sin error, si no es ok retorna url:false si es ok url:true
		if response.StatusCode != http.StatusOK {
			results <- fmt.Sprintf("workerid: %d para url %s resulta False \n", workerId, url)
		} else {
			results <- fmt.Sprintf("workerid: %d para url %s resulta True \n", workerId, url)
		}
	}
}

// solo llama a la función de verificar sitios con un slice de urls
func main() {
		// declara un array de urls para chequear
		var websites = []string {
			"http://ort.edu.uy",
			"http://google.com",
			"http://github.com",
			"http://arqsoft.com",
			"http://netflix.com",
			"http://instagram.com",
			"http://ingsoft.gaston.com",
		}

	fmt.Printf("*****comienzo *****\n")

	CheckWebsites(websites)

	fmt.Printf("***** FIN *****")
}