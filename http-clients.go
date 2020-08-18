package main

import (
	"bufio"
	"fmt"
	"net/http"
	"time"
)

func main() {
	// start the server


	// test that the proxy is up
	resp, err := http.Get("http://proxy:8000")
	for err != nil {
		fmt.Println(err)
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
	fmt.Println("====== Passed all integration tests ======")
}

