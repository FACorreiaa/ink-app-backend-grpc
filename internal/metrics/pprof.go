package metrics

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func InitPprof() {
	//http.HandleFunc("/debug/pprof/", pprof.Index)
	//http.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	//http.HandleFunc("/debug/pprof/profile?debug=1", pprof.Profile)
	//http.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	//http.HandleFunc("/debug/pprof/trace", pprof.Trace)
	//http.HandleFunc("/debug/pprof/allocs", handlerFunc(pprof.Handler("allocs")))
	//http.HandleFunc("/debug/pprof/block", handlerFunc(pprof.Handler("block")))
	//http.HandleFunc("/debug/pprof/goroutine", handlerFunc(pprof.Handler("goroutine")))
	//http.HandleFunc("/debug/pprof/heap", handlerFunc(pprof.Handler("heap")))
	//http.HandleFunc("/debug/pprof/mutex", handlerFunc(pprof.Handler("mutex")))
	//http.HandleFunc("/debug/pprof/threadcreate", handlerFunc(pprof.Handler("threadcreate")))

	go func() {
		log.Println("Running pprof!")

		fmt.Println(http.ListenAndServe(":6060", nil))
	}()
}

func handlerFunc(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}
}

//func chiWrapper(fn http.HandlerFunc) http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		ctx := r.Context()
//		fn(w, r.WithContext(ctx))
//	}
//}
