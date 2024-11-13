package main

import (
	"fmt"
	"net/http"
)

func main() {
	{
		fmt.Println("Dummy is running")
		//go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "Hello, Dummy!")
		})

		if err := http.ListenAndServe(":80", nil); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
		}
		//}()

	}
}
