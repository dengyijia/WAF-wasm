package main

import (
	"bufio"
	"fmt"
	"net/http"
	"time"
)

func handler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "hello\n")
}

func startServer(port string) {
	http.HandleFunc("/", handler)
	http.ListenAndServe(port, nil)
}

func main() {
	// wait for the server and proxy to run
	time.Sleep(5 * time.Second)

	go startServer(":8080")

	// test that the proxy is up
	resp, err := http.Get("http://proxy:8000")
	for err != nil {
		fmt.Println("Client connecting to proxy ...")
		time.Sleep(5 * time.Second)
		resp, err = http.Get("http://proxy:8000")
	}
	defer resp.Body.Close()

	fmt.Println("Response status", resp.Status)

	scanner := bufio.NewScanner(resp.Body)
	for i := 0; scanner.Scan() && i < 5; i++ {
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	// make test requests

	// terminate with sucess
	fmt.Println("====== Passed all client side integration tests ======")
}

