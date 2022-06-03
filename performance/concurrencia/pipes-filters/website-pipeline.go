/*
	este ejemplo utiliza canales para sincronizar las gorutinas y muestra el uso
	del patrón pipeline y de sincronización Fan-out (varios gorutinas leyendo de un mismo canal), y hace un merge de los
	resultados en un nuevo canal
		
	
			   / -canal1 ----> gr1 (feedWebsites) ----\
	generador / - canal2 ----> gr2 (feedWebsites) ---> merge() --> imprimir

	en este ejemplo se utilizan el fan out y el fan in para mostrar ambos, pero no simpre es necesario utilizar ambos o ninguno
	el beneficio del fan out es que se procesa un feed en varias gorutinas concurrentes.

	ejemplo basado en 
		https://go.dev/blog/pipelines 	y el libro Concurrency in Go por Katherine Cox-Buday

	puede ejecutarlo con
		go run . para ejecutar el main o
		go test -bench=. que ejecuta un benchamrk para ver los tiempo que demora la ejecución
		concurrente

	al ejecutar prestar atención al orden en que se despliega el workerId o el url y
	a la cantidad de segundos que duró la ejecución (con go test -bench)

	TO DO - agregar manejo de errores

*/

package main

import (
	"fmt"
	"net/http"
	"sync"
)


// función extraida de checkWebsite para hacerla mas legible a checkWebsite
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


// esta función es la que llama a las goroutinas que checkean los urls en paralelo 
// usa el select que en caso de que haya algo en el canal de in lo procesa (callhead) y lo
// empuja al canal out
// el select chequea el done y en caso que venga algo en ese canal termina prolijamente la ejecución de 
// la gorutina
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

// función que mergea los canales de salida de las gorutinas que usa
// checkWebsite
// ver pipeline ( WebsiteStatusChecker()) para ver el caso de uso

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


/* esta función arma el pipeline
	crear un canal de control para avisar a las gorutinas que terminen
	llama a feedWebsites que es un generador de datos (convierte los datos del slice en un stream
	que empuja por el canal in <-string)
	lo que viene en el canal se abre en dos gorutinas (checkWebsite) y luego el resultado de los canales
	se mergea en un único canal para poder imprimir. 
	recodar que los pipelines en general tienen un source (el generador), varios filtros y que en general terminan en un sink que 
	es donde se juntan todas las ramas o donde termina el pipeline

*/
func WebsiteStatusChecker()	{

	//se crea un canal para comunicar que las gorutinas  terminen su trabajo y salgan prolijamente
	done := make(chan struct{})
	defer close(done)

	// empuja por el canal in los urls
	in := feedWebsites(done)

	// se crear 2 gorutinas que cada una saca de in y escribe a su propio canal de salida

	firstFeeder := checkWebsite(done, in)
	secondFeeder := checkWebsite(done, in)

	// función que mergea lo que pasa en los dos canales de entrada en uno de salida 
	out := merge(done, firstFeeder, secondFeeder)

	// se imprime el resultado leyendo del canal mergeado.
	for n := range out {
		fmt.Printf(n)
	}
	
}



// solo llama a la función de verificar sitios con un slice de urls, se separó para poder invocar WebsiteStatusChecker(); 
// desde los tests
func main() {
	// declara un array de urls para chequear

	fmt.Printf("*****comienzo *****\n")

	WebsiteStatusChecker();

	fmt.Printf("***** FIN *****")
}
