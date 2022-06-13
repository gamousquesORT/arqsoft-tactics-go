/*
	este ejemplo utiliza canales para sincronizar las gorutinas y muestra el uso
	del patrón pipeline y Fan-out (varios gorutinas leyendo de un mismo canal), y hace un merge de los
	resultados en un nuevo canal (fan -in)


			   / -canal1 ----> gr1 (checkWebsite) ----\        /--> gr1 (convertResultaToUpperCase)---\
	producer / - canal2 ----> gr2 (checkWebsite) ---> merge() / --> gr2 (convertResultaToUpperCase)--->merge()-->sink (imprimir)

	en el ejemplo se utilizan el fan out y el fan in para mostrar ambos, pero no siempre es necesario utilizarlos. 
	el beneficio del fan out es que se procesa un feed en varias gorutinas concurrentes. se lanzan tantas gorutinas de cada paso como 
	CPUs disponibles tengamos.

	también se agregó el uso de Contexto para cancelar (la opción de cancel esta comentada en el sink)

	ejemplo basado en
		https://go.dev/blog/pipelines , https://medium.com/amboss/applying-modern-go-concurrency-patterns-to-data-pipelines-b3b5327908d4
		y el libro Concurrency in Go por Katherine Cox-Buday

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
	"errors"
	"net/http"
	"strings"
	"context"
	"log"
	"runtime"
	"sync"
)

// función extraída de checkWebsite para hacerla mas legible a checkWebsite
func callHead(url string) (string, error) {
	var ret string

	// el que sigue es el mismo código que en el ejemplo secuencial
	response, err := http.Head(url)
	if err != nil {
		 ret = fmt.Sprintf("El url %s no existe \n", url)
	} else {
		// ok Head sin error, si no es ok retorna url:false si es ok url:true
		if response.StatusCode != http.StatusOK {
			ret = fmt.Sprintf("El url %s NO responde OK código %d \n", url, response.StatusCode)
		} else {
			ret = fmt.Sprintf("El url %s responde OK \n", url)
		}
	}
	return ret, err
}

// esta función es la que llama a la goroutina que checkean los urls  
// usa el select que en caso de que haya algo en el canal de in lo procesa (callhead) y lo
// empuja al canal out
// el select chequea el done y en caso que venga algo en ese canal termina prolijamente la ejecución de 
// la gorutina
// si un url no existe lo pasa al canal de errores para que lo maneje el sink
func checkWebsite(ctx context.Context, in <-chan string) (<-chan string, <-chan error, error) {

	out := make(chan string)
	errorChannel := make(chan error)
	
	go func() {
		defer close(out)
		defer close(errorChannel)

		for url := range in {
			select {
			case <- ctx.Done():
				return	
			default:
				url, err := callHead(url)
				if err != nil {
					errorChannel <- errors.New(url)
				} else {
					out <- url
				}
			
			}
		}
	}()
	
	return out, errorChannel, nil

}

//esta gorutina convierte un string a mayúscula
func convertResultaToUpperCase(ctx context.Context, in <-chan string) (<-chan string, <-chan error, error) {

	out := make(chan string)
	errorChannel := make(chan error)

	go func() {
		defer close(out)
		defer close(errorChannel)

		for result := range in {
			select {
			case <- ctx.Done():
				return
			default:
				out <- strings.ToUpper(result)
			}
		}
	}()
	
	return out, errorChannel, nil

}

// funciona generadora es la fuente de datos que alimenta el stream que va a pasar por el pipeline. 
func producer(ctx context.Context) (<-chan string, error) {
	out := make(chan string)
	
	var websites = []string{
		"http://ort.edu.uy",
		"http://google.com",
		"http://github.com",
		"http://arqsoft.com",
		"http://netflix.com",
		"http://instagram.com",
		"http://ingsoft.gaston.com",
		"http://gitlab.com",
		"http://gaston.arq.com",
	}
	go func ()  {
		defer close(out)
		for _, ws := range websites {
			select {
			case <-ctx.Done():
				return
			default:
				out <- ws
			}
		}
	}()
	
	return out, nil
}

// función sink que despliega los url procesados. en caso de que venga algo en el canal de errores cancela todos los pipelines
// (ver cancel comentado para hacer el log y no terminar todo el pipeline)
func sink(ctx context.Context, cancel context.CancelFunc,values <-chan string, errors <-chan error) {
	for {
		select {
		case <-ctx.Done():
			log.Print(ctx.Err().Error())
			return
			
		case err, ok := <-errors:
		if ok {
       //cancel()
				log.Print(err.Error())
			}
	
		case val, ok := <-values:
			if ok {
				log.Printf("sink: %s", val)
			} else {
				log.Print("done")
				return
			}
		}
	}
}

// mergea canales de entrada a uno de salida
func mergeStringChans(ctx context.Context, cs ...<-chan string) <-chan string {
	var wg sync.WaitGroup
	out := make(chan string)

	output := func(c <-chan string) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// mergea los canales de error en uno solo
func mergeErrorChans(ctx context.Context, cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	output := func(c <-chan error) {
		defer wg.Done()
		for n := range c {
			select {
			case out <- n:
			case <-ctx.Done():
				return
			}
		}
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

/* esta función arma el pipeline
	crear un context para avisar mediante el canal done a las gorutinas que terminen
	llama a generator que es el generador de datos (convierte los datos del slice en un stream
	que empuja por el canal in <-string)
	lo que viene en el canal se abre en varias gorutinas (checkWebsite) y luego el resultado de los canales
	se mergea en un único canal para poder pasarla al siguiente filtro (convertResultaToUpperCase), finalmente se pasa el sink que es 
	el filtro que imprime. 
	recodar que los pipelines en general tienen un source (el generador), varios filtros y que en general terminan en un sink que 
	es donde se juntan todas las ramas o donde termina el pipeline

*/
func WebsiteStatusChecker()	{

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()


	//se crea un canal para comunicar que las gorutinas  terminen su trabajo y salgan prolijamente
	done := make(chan struct{})
	defer close(done)

	fmt.Printf("---> llamar producer *****\n")
	// empuja por el canal in los urls
	in, err := producer(ctx)
	if err != nil {
		log.Fatal(err)
	}


	// fan out stage1
	stage1Channels := []<-chan string{}
	errors := []<-chan error{}
	fmt.Printf("---> for stage1 *****\n")
	for i := 0; i < runtime.NumCPU(); i++ {
		fmt.Printf("---> for stage1 (%d) *****\n",i)
		websiteCheckChannel, websiteCheckErrors, err := checkWebsite(ctx, in)
		if err != nil {
			log.Fatal(err)
		}
		stage1Channels = append(stage1Channels, websiteCheckChannel)
		errors = append(errors, websiteCheckErrors)
	}

	// fan in - stage1
	stage1Merged := mergeStringChans(ctx, stage1Channels...)

	// fan out stage2
	fmt.Printf("---> for stage2 *****\n")
	stage2Channels := []<-chan string{}

	for i := 0; i < runtime.NumCPU(); i++ {
		fmt.Printf("---> for stage2 (%d) *****\n",i)
		toUpperCaseChannel, toUpperErrors, err := convertResultaToUpperCase(ctx, stage1Merged)
		if err != nil {
			log.Fatal(err)
		}
		stage2Channels = append(stage2Channels, toUpperCaseChannel)
		errors = append(errors, toUpperErrors)
	}

	stage2Merged := mergeStringChans(ctx, stage2Channels...)

	// fan in - stage2
	errorsMerged := mergeErrorChans(ctx, errors...)
	fmt.Printf("---> sink *****\n")
	sink(ctx, cancel, stage2Merged, errorsMerged)

}
	


// solo llama a la función de verificar sitios con un slice de urls, se separó para poder invocar WebsiteStatusChecker(); 
// desde los tests
func main() {
	// declara un array de urls para chequear

	fmt.Printf("*****comienzo *****\n")

	WebsiteStatusChecker();

	fmt.Printf("***** FIN *****")
}
