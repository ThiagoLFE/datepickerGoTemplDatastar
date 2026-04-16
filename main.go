package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", RenderDatepicker)
	fs := http.FileServer(http.Dir("."))
	mux.Handle("GET /datastar.js", fs)
	mux.Handle("GET /utils.js", fs)
	mux.Handle("GET /datepicker.js", fs)
	mux.Handle("GET /layout.css", fs)
	mux.Handle("GET /style.css", fs)

	fmt.Println("server on...")
	http.ListenAndServe(":8080", mux)
}

func RenderDatepicker(w http.ResponseWriter, req *http.Request) {
	Page(InputDate("calendario")).Render(req.Context(), w)
}
