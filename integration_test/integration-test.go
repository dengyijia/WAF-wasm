package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
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
	proxyURL string

	// the port that the test server will be listening at
	serverPort string

	// the expected message received from the test server if
	// the request has been successful
	serverMessage string

	// the server that will be responding requests from the proxy
	server http.Server

	// the channels for communication between the server
	// and the client. Received requests will be pushed through these
	// channels by the server for the client to verify
	requestReceived    chan *http.Request
	bodyStringReceived chan string

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
	path       string
	pathParams url.Values
	headers    Map
	cookies    Map
	body       url.Values

	// the expected sqli detection result for the proxy
	// if true, the request contains sqli
	sqli      bool
	sqli_body bool
}

/*
* Initiate a new test suite with a given proxyURL
 */
func NewTestSuite(proxyURL string) TestSuite {
	return TestSuite{
		proxyURL:           proxyURL,
		serverPort:         ":8080",
		serverMessage:      "request received",
		requestReceived:    make(chan *http.Request),
		bodyStringReceived: make(chan string),
		requests:           map[string]TestRequest{},
	}
}

/*
* Run all requests in a test suite
 */
func (suite TestSuite) RunRequests() {
	suite.StartServer()
	suite.CheckProxyConnection()
	for name, test := range suite.requests {
		suite.RunTestRequest(name, test)
	}
	suite.CloseServer()
}

/*
* Initiate a new test request with default content
 */
func NewTestRequest() TestRequest {
	return TestRequest{
		path:       "/",
		pathParams: url.Values{"path-key-0": {"path-val-0"}},
		headers:    Map{"header-key-0": "header-val-0"},
		cookies:    Map{"cookie-key-0": "cookie-val-0"},
		body:       url.Values{"body-key-0": {"body-val-0"}},
		sqli:       false,
		sqli_body:  false,
	}
}

/*
* Run a given test in a test suite
* assuming the server and the proxy has been running
 */
func (suite TestSuite) RunTestRequest(testName string, test TestRequest) {
	result := suite.Eval(test)
	if !result {
		fmt.Println("TEST FAILED: " + testName)
		fail()
	}
	fmt.Println("TEST PASSED: " + testName)
}

/*
* Initialize and Start the test server
 */
func (suite TestSuite) StartServer() {
	// set path handler functions
	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		go func() {
			suite.requestReceived <- r
		}()
		go func() {
			suite.bodyStringReceived <- string(body)
		}()
		fmt.Fprintf(w, suite.serverMessage)
	})
	serveMux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		suite.server.Close()
	})

	// start server
	suite.server = http.Server{
		Addr:    suite.serverPort,
		Handler: serveMux,
	}
	go func() {
		err := suite.server.ListenAndServe()
		if err != http.ErrServerClosed {
			fmt.Println("Server shutdown unexpectedly:", err)
			fail()
		}
	}()
	fmt.Println("Server started")
}

/*
* Close the test server
 */
func (suite TestSuite) CloseServer() {
	_, err := http.Get(suite.proxyURL + "/shutdown")
	if err != nil {
		fmt.Println("Server shutdown failed:", err)
		fail()
	}
	close(suite.requestReceived)
	close(suite.bodyStringReceived)
}

/*
* Test if the proxy and the test server has been properly connected
 */
func (suite TestSuite) CheckProxyConnection() {
	// wait and test that the proxy is up
	fmt.Println("Check proxy connection")
	time.Sleep(1 * time.Second)
	response, err := http.Get(suite.proxyURL)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Client connecting to proxy ...")
		suite.CheckProxyConnection()
		return
	}
	defer response.Body.Close()

	// check that proxy reply indeed comes from server
	responseBody, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != 200 || string(responseBody) != suite.serverMessage {
		fmt.Println("Proxy connection failed")
		fmt.Println("Received response", response.Status, string(responseBody))
		fail()
	}
	<-suite.requestReceived
	<-suite.bodyStringReceived
	fmt.Println("Client connected to server through proxy")
}

/*
* evaluate if a given request is handled properly by the proxy
* true if the test passed, false if not
 */
func (suite TestSuite) Eval(test TestRequest) bool {
	// initialize request
	requestBody := test.body.Encode()
	requestPathParams := test.pathParams.Encode()
	request, _ := http.NewRequest(
		"POST",
		suite.proxyURL+test.path+"?"+requestPathParams,
		strings.NewReader(requestBody),
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

	// wait for the received request to send through
	time.Sleep(100 * time.Millisecond)

	// get request from server
	var received *http.Request
	var receivedBody string
	select {
	case received = <-suite.requestReceived:
	default:
		received = nil
	}
	select {
	case receivedBody = <-suite.bodyStringReceived:
	default:
		receivedBody = ""
	}

	// verify the request and the response
	if test.sqli {
		// if the request contains SQL injection, it should be blocked
		// if the SQL injection is in body, the server should receive request with empty body
		// if the SQL injection is in header, the server should never receive any request
		if test.sqli_body {
			if receivedBody != "" {
				fmt.Println("Request with SQL injection in body was not blocked")
				fmt.Println(receivedBody)
				return false
			}
		} else if received != nil {
			fmt.Println("Request with SQL injection was not blocked")
			fmt.Println(response.Status)
			return false
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
				fmt.Println("EXPECTED: ", key, " -> ", val)
				fmt.Println("ACTUAL: ", key, " -> ", received.Header.Get(key))
				return false
			}
		}
		for key, val := range test.cookies {
			cookieSent, err := received.Cookie(key)
			if err != nil || cookieSent.Value != val {
				fmt.Println("Request without SQL injection has cookie altered by proxy")
				fmt.Println("EXPECTED: ", cookieSent)
				return false
			}
		}
		if receivedBody != requestBody {
			fmt.Println("Request without SQL injection has body altered by proxy")
			fmt.Println("EXPECTED: ", requestBody)
			fmt.Println("ACTUAL: ", receivedBody)
			return false
		}

		// the client should receive a response from the server with status 200 Okay
		responseBody, _ := ioutil.ReadAll(response.Body)
		if response.StatusCode != 200 || string(responseBody) != suite.serverMessage {
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
