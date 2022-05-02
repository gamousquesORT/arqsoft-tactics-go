/*
	este ejemplo es el mismo que sync_v1 pero utilizando goroutines como funciones anónimas.
		ej.  	go func (u string, workerId int)  {

	Go cuenta con dos grupos de mecanismos de concurrencia.
	el paquete syn que permite forkear goroutines y hacer join de su ejecución
	sync también cuenta con semáforos (mutex y variantes) para proteger variables o "memoria" que
	comparten las goroutines. En este caso las gorrutinas no comparten información.

	Por ejemplo si quisiéramos utilizar una slice o map para guardar los resutados que se
	despliegan cada llamada a CheckOneWebsite(...) esa estructura deberia estar protegida
	por un sync.Mutex

	Este ejemplo solamente utiliza el mecanismo de fork y join que se conoce como WaitGroup
	el WaitGroup permite
	  - llevar el contardor de goroutinas a sincronizar Add(n)
	  - decrementar el contador cuando una goroutina termina Done()
	  - esperar que todas las goroutinas en ejecución terminen Wait()

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
)

// esta función es la que llama a las goroutinas y las sincroniza
// utilizando sync.WaitGroups
func CheckWebsite(in <-chan string) <-chan string {

	out := make(chan string)


	go func() {

		for url := range in {

			// el que sigue es el mismo código que en el ejemplo secuencial
			response, err := http.Head(url)
			if err != nil {
				out <- fmt.Sprintf("El url %s no existe \n", url)
			} else {
				// ok Head sin error, si no es ok retorna url:false si es ok url:true
				if response.StatusCode != http.StatusOK {
					out <- fmt.Sprintf("El url %s NO responde OK \n", url)
				} else {
					out <- fmt.Sprintf("El url %s responde OK \n", url)
				}
			}
		}
		close(out)
	}()


		return out

}

func FeedWebsites() <-chan string {
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
		for _, ws := range websites {
			out <- ws
		}
		close(out)
		
	}()

	return out
}

func WebsiteStatusChecker()	{

	for n:= range CheckWebsite(FeedWebsites()) {
		fmt.Printf("%s", n)
	}
}


// solo llama a la función de verificar sitios con un slice de urls
func main() {
	// declara un array de urls para chequear

	fmt.Printf("*****comienzo *****\n")

	WebsiteStatusChecker();

	fmt.Printf("***** FIN *****")
}
