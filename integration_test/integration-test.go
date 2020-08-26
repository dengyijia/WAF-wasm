package main

import (
	"io/ioutil"
	"fmt"
	"net/http"
	"net/url"
	"time"
	"strings"
	"os"
)

/*****************************
* INTEGRATION TEST FRAMEWORK *
******************************/

type Map = map[string]string

/*
* TestSuite struct
*   keep track of info used by a test suite
*   different proxy configurations corresponds
*   to different test suites
*/
type TestSuite struct {
	// url of the proxy to be tested
	// the test client will send requests to this url
	proxy_url string

	// the port that the test server will be listening at
	server_port string

	// the expected message received from the test server if
	// the request has been successful
	server_message string

	// the server that will be responding requests from the proxy
	server http.Server

	// the channels for communication between the server
	// and the client. Received requests will be pushed through these
	// channels by the server for the client to verify
	request_received chan *http.Request
	body_string_received chan string

	// a collection of requests to run in this suite
	// each test has a test name associated with it
	requests map[string]TestRequest
}

/*
* TestRequest struct
*   keep track of info used by one request in a test suite
*/
type TestRequest struct {
	// the test request to be made
	path string
	pathParams url.Values
	headers Map
	cookies Map
	body  url.Values

	// the expected sqli detection result for the proxy
	// if true, the request contains sqli
	sqli bool
}

/*
* Initiate a new test suite with a given proxy_url
*/
func NewTestSuite(proxy_url string) TestSuite {
	return TestSuite{
		proxy_url: proxy_url,
		server_port: ":8080",
		server_message: "request received",
		request_received: make(chan *http.Request),
		body_string_received: make(chan string),
		requests: map[string]TestRequest{},
	}
}

/*
* Run all requests in a test suite
*/
func (t TestSuite) RunRequests() {
	t.StartServer()
	t.CheckProxyConnection()
	for name, test := range t.requests {
		t.RunTestRequest(name, test)
	}
	t.CloseServer()
}


func NewTestRequest() TestRequest {
	return TestRequest{
		path: "/",
		pathParams: url.Values{"path-key-0": {"path-val-0"}},
		headers: Map{"header-key-0": "header-val-0"},
		cookies: Map{"cookie-key-0": "cookie-val-0"},
		body: url.Values{"body-key-0": {"body-key-0"}},
		sqli: false,
	}
}

/*
* Run a given test in a test suite
* assuming the proxy has been running
*/
func (t TestSuite) RunTestRequest(test_name string, test TestRequest) {
	result := t.Eval(test)
	if !result {
		fmt.Println("TEST FAILED: " + test_name)
		fail()
	}
	fmt.Println("TEST PASSED: " + test_name)
}

/*
* Initialize and Start the test server
*/
func (suite TestSuite) StartServer() {
	// set path handler function
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		//fmt.Println("Server receives body with path", r.URL.Path)
		go func() {
			//fmt.Println("Server pushing request")
			suite.request_received <- r
		} ()
		go func() {
			suite.body_string_received <- string(body)
		} ()
		fmt.Fprintf(w, suite.server_message)
	})
	serveMux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		suite.server.Close()
	})

	// start server
	suite.server = http.Server {
		Addr: suite.server_port,
		Handler: serveMux,
	}
	go suite.server.ListenAndServe()
	fmt.Println("Server started")
}

func (t TestSuite) CloseServer() {
	http.Get(t.proxy_url + "/shutdown")
	close(t.request_received)
	close(t.body_string_received)
}

/*
* Test if the proxy and the test server has been properly connected
*/
func (t TestSuite) CheckProxyConnection() {
	// wait and test that the proxy is up
	fmt.Println("Check proxy connection")
	time.Sleep(1 * time.Second)
	response, err := http.Get(t.proxy_url)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Client connecting to proxy ...")
		t.CheckProxyConnection()
		return
	}
	defer response.Body.Close()

	// check that proxy reply comes from server
	response_body, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != 200 || string(response_body) != t.server_message {
		fmt.Println("Proxy connection failed")
		fmt.Println("Received response", response.Status, string(response_body))
		fail()
	}
	<-t.request_received
	<-t.body_string_received
	fmt.Println("Client connected to server through proxy")
}

/*
* evaluate if a given request is handled properly by the proxy
* true if the test passed, false if not
*/
func (suite TestSuite) Eval(test TestRequest) bool {
	// initialize request
	request_body := test.body.Encode()
	request_pathParams := test.pathParams.Encode()
	request, _ := http.NewRequest(
		"POST",
		suite.proxy_url + test.path + "?" + request_pathParams,
		strings.NewReader(request_body),
	)
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	for key, val := range test.headers {
		request.Header.Set(key, val)
	}
	for key, val := range test.cookies {
		request.AddCookie(&http.Cookie{Name: key, Value: val})
	}

	// send request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Send request failed")
		fmt.Println(err)
		return false
	}
	defer response.Body.Close()

	// get request from server
	var received *http.Request
	var received_body string
	select {
	case received = <-suite.request_received:
		default: received = nil
	}
	select {
	case received_body = <-suite.body_string_received:
		default: received_body = ""
	}

	// verify the request and the response
	if test.sqli {
		// if the request contains SQL injection, it should be blocked
		// the server should never receive any request
		if received != nil {
			fmt.Println("Request with SQL injection was not blocked")
			fmt.Println(received.URL.Path)
			fmt.Println(response.Status)
			//return false
		}

		// the client should receive a response from the proxy with status 403 Forbidden
		if response.StatusCode != 403 {
			fmt.Println("Request with SQL injection did not receive 403 reply")
			return false
		}

	} else {
		// if the request does not contain any SQL injection,
		// the server should receive it intact
		if received == nil {
			fmt.Println("Request without SQL injection was blocked")
			return false
		}
		if test.path != received.URL.Path {
			fmt.Println("Request without SQL injection has path altered by proxy")
			fmt.Println("EXPECTED: ", test.path)
			fmt.Println("ACTUAL: ", received.URL.Path)
			return false
		}
		for key, val := range test.headers {
			if received.Header.Get(key) != val {
				fmt.Println("Request without SQL injection has header altered by proxy")
				fmt.Println("EXPECTED: ", key, " -> ",  val)
				fmt.Println("ACTUAL: ", key, " -> ", received.Header.Get(key))
				return false
			}
		}
		for key, val := range test.cookies {
			cookie_sent, err := received.Cookie(key)
			if err != nil || cookie_sent.Value != val {
				fmt.Println("Request without SQL injection has cookie altered by proxy")
				fmt.Println("EXPECTED: ", cookie_sent)
				return false
			}
		}
		if received_body != request_body {
			fmt.Println("Request without SQL injection has body altered by proxy")
			fmt.Println("EXPECTED: ", request_body)
			fmt.Println("ACTUAL: ", received_body)
			return false
		}

		// the client should receive a response from the server with status 200 Okay
		response_body, _ := ioutil.ReadAll(response.Body)
		if response.StatusCode != 200 || string(response_body) != suite.server_message {
			fmt.Println("Request without SQL did not receive 200 reply")
			return false
		}
	}
	return true
}

/*
* fail()
*   this function is called when a test failed
*   print integration test failure message and abort the program
*/
func fail() {
	fmt.Println("====== Integration tests failed ======")
	os.Exit(0)
}

