package metrics

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func InitPprof() {
	go func() {
		log.Println("\nRunning pprof!")

		fmt.Println(http.ListenAndServe(":6060", nil))
	}()
}
