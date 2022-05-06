/*
	este ejemplo utiliza canales para sincronizar las gorutinas y muestra el uso
	del patrón pipeline y de sincronización Fan-out (varios gorutinas leyendo de un mismo canal), 
	fan-in (un funcion lee de varias fuentes y las multiplexa en un canal)


		https://go.dev/blog/pipelines

	puede ejecutarlo con
		go run . para ejecutar el main o
		go test -bench=. que ejecuta un benchamrk para ver los tiempo que demora la ejecución
		concurrente

	al ejecutar prestar atención al orden en que se despliega el workerId o el url y
	a la cantidad de segundos que duró la ejecución (con go test -bench)


    el ejemplo se adapta de https://github.com/quii/learn-go-with-tests/tree/main/concurrency
	y el libro Concurrency in Go por Katherine Cox-Buday
*/

package main

import (
	"fmt"
	"net/http"
	"sync"
)

// esta función es la que llama a las goroutinas y las sincroniza
// utilizando sync.WaitGroups
func checkWebsite(done <- chan struct {}, in <-chan string) <-chan string {

	out := make(chan string)
	go func() {
		defer close(out)
		for url := range in {
			select {
			case out <- callHead(url):
			case <- done:
				return	
			}
		}
	}()
	
	return out

}

func callHead(url string) string {
	var ret string

	// el que sigue es el mismo código que en el ejemplo secuencial
	response, err := http.Head(url)
	if err != nil {
		 ret = fmt.Sprintf("El url %s no existe \n", url)
	} else {
		// ok Head sin error, si no es ok retorna url:false si es ok url:true
		if response.StatusCode != http.StatusOK {
			ret = fmt.Sprintf("El url %s NO responde OK \n", url)
		} else {
			ret = fmt.Sprintf("El url %s responde OK \n", url)
		}
	}
	return ret
}

func merge(done <-chan struct{}, cs... <-chan string) <-chan string {
    var wg sync.WaitGroup
    out := make(chan string)

	// esta gorutina saca de los canales de entrada y pasa lo recibido a un canal único (out)
	// notar que en 2do argumento de merge es un colección de canales
	// se usa el waitgroup para saber cuantos canales están aportando al merge
    output := func(c <-chan string) {
		defer wg.Done()
        for n := range c {
            select {
            case out <- n:
            case <-done:
                return
            }
        }
    }

	wg.Add(len(cs))

	// se ejecuta la gorutina anterior pasando cada canal
    for _, c := range cs {
        go output(c)
    }

	// se ejecuta una gorutina que cierra el canal último cundo 
	// las anteriores teriminaron (los sabe por el <- done del select y el wait)
    go func() {
        wg.Wait()
        close(out)
    }()

    return out

}


func feedWebsites(done <- chan struct{}) <-chan string {
	out := make(chan string)
	
	var websites = []string{
		"http://ort.edu.uy",
		"http://google.com",
		"http://github.com",
		"http://arqsoft.com",
		"http://netflix.com",
		"http://instagram.com",
		"http://ingsoft.gaston.com",
	}
	go func ()  {
		defer close(out)
		for _, ws := range websites {
			select {
			case out <- ws:
			case <-done:
				return
			}
		}
	}()
	
	return out
}

func WebsiteStatusChecker()	{

	//se crea un canal para comunicar que las gorutinas fueron terminando su trabajo
	done := make(chan struct{})
	defer close(done)


	in := feedWebsites(done)

	firstFeeder := checkWebsite(done, in)
	secondFeeder := checkWebsite(done, in)

	// función que mergea lo que pasa en los dos canales de entrada en uno de salida 
	out := merge(done, firstFeeder, secondFeeder)

	for n := range out {
		fmt.Printf(n)
	}
	
}


// solo llama a la función de verificar sitios con un slice de urls
func main() {
	// declara un array de urls para chequear

	fmt.Printf("*****comienzo *****\n")

	WebsiteStatusChecker();

	fmt.Printf("***** FIN *****")
}
