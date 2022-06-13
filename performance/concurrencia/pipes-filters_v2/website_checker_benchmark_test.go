// para probar el benchmark 
// go test -bench website_checker_benchmark_test.go
//https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go

package main

import("testing")



	  // benchmark de llamadas secuenciales
func BenchmarkWebsiteChecker(b *testing.B) {

	// declara un slice de urls para chequear
	   // Demora tenga paciencia!
	   // hace el benchmark hasta que N que es un valor que el runtime determina para que sea significativo
	   // Demora tenga paciencia!
	   //
	for i:=1 ; i < b.N; i++ {
		WebsiteStatusChecker()
	}
}