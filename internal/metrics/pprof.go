package metrics

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	_ "net/http/pprof"
)

func InitPprof() {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile?debug=1", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	mux.HandleFunc("/debug/pprof/allocs", handlerFunc(pprof.Handler("allocs")))
	mux.HandleFunc("/debug/pprof/block", handlerFunc(pprof.Handler("block")))
	mux.HandleFunc("/debug/pprof/goroutine", handlerFunc(pprof.Handler("goroutine")))
	mux.HandleFunc("/debug/pprof/heap", handlerFunc(pprof.Handler("heap")))
	mux.HandleFunc("/debug/pprof/mutex", handlerFunc(pprof.Handler("mutex")))
	mux.HandleFunc("/debug/pprof/threadcreate", handlerFunc(pprof.Handler("threadcreate")))

	go func() {
		log.Println("\nRunning pprof!")

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
