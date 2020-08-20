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
/*
* type aliases for request components
*/
type Header = map[string]string
type Cookie = map[string]string
type Body = map[string]string

/*
* Test struct
*   keep track of info used by a test suite
*/
type Test struct {
  // the proxy url for the test client to send requests to
  proxy_url string

  // the expected message received from the test server if
  // the request has been successful
  server_message string
  server_port string

  // the channel for communication between the server goroutine
  // and the client. Received requests will be pushed through this
  // channel by the server for the client to verify
  request_received chan *http.Request
  body_string_received chan string
}

func (t Test) StartServer() {
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    go func() {
      t.request_received <- r
    }()
    go func() {
      body, _ := ioutil.ReadAll(r.Body)
      fmt.Println("server body: ", string(body))
      t.body_string_received <- string(body)
    }()
    fmt.Fprintf(w, t.server_message)
  })
  go http.ListenAndServe(t.server_port, nil)
  fmt.Println("Server started")
}

func (t Test) CheckProxyConnection() {
  // wait and test that the proxy is up
  time.Sleep(2 * time.Second)
  response, err := http.Get(t.proxy_url)
  if err != nil {
    fmt.Println("Client connecting to proxy ...")
    t.CheckProxyConnection()
    return
  }
  defer response.Body.Close()

  // check that proxy reply comes from server
  response_body, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 || string(response_body) != t.server_message {
    fmt.Println("Proxy connection failed")
    fail()
  }
  <-t.request_received
  <-t.body_string_received
  fmt.Println("Client connected to server through proxy")
}

/*
* test()
*   test if a given request is handled properly by the proxy
* Input:
*   path, header, cookie, body: components of the request to be made
*   sqli: true if the request contains a sql injection, false if not
*   proxy_url: url of the proxy to send request to
*   request_received: a channel shared with the server, the request received
*     by the server will be pushed to this channel
* Output:
*   true if the test passed, false if not
*/
func (t Test) Test(path string, headers Header, cookies Cookie, body Body, sqli bool) bool {
  // set request path, body, header, cookie
  request_body_url := url.Values{}
  for key, val := range body {
    request_body_url.Set(key, val)
  }
  request_body := request_body_url.Encode()
  request, _ := http.NewRequest("POST", t.proxy_url + path, strings.NewReader(request_body))
  for key, val := range headers {
    request.Header.Set(key, val)
  }
  for key, val := range cookies {
    request.AddCookie(&http.Cookie{Name: key, Value: val})
  }

  for len(t.request_received) > 0 {
    <-t.request_received
  }
  for len(t.body_string_received) > 0 {
    <-t.body_string_received
  }

  // send request
  client := &http.Client{}
  response, err := client.Do(request)
  if err != nil {
    fmt.Println("Send request failed")
    return false
  }
  defer response.Body.Close()

  time.Sleep(2 * time.Second)
  // check results of the request
  var received *http.Request
  var received_body string
  select {
    case received = <-t.request_received:
    default: received = nil
  }
  select {
    case received_body = <-t.body_string_received:
    default: received_body = ""
  }

  if sqli {
    // if the request contains SQL injection, it should be blocked
    // the server should never receive any request
    if received != nil {
      fmt.Println("Request with SQL injection was not blocked")
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
    for key, val := range headers {
      if received.Header.Get(key) != val {
        fmt.Println("Request without SQL injection has header altered by proxy")
	fmt.Println("EXPECTED: ", key, " -> ",  val)
	fmt.Println("ACTUAL: ", key, " -> ", received.Header.Get(key))
        return false
      }
    }
    for _, cookie_received := range received.Cookies() {
      cookie_sent, found := cookies[cookie_received.Name]
     if found == false || cookie_sent != cookie_received.Value {
	fmt.Println("Request without SQL injection has cookie altered by proxy")
	fmt.Println("EXPECTED: ", cookie_received.Name, cookie_sent)
	fmt.Println("ACTUAL: ", cookie_received.Name, cookie_received.Value)
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
    if response.StatusCode != 200 || string(response_body) != t.server_message {
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

func run(test_name string, result bool) {
  if !result {
    fmt.Println("TEST FAILED: " + test_name)
    fail()
  }
  fmt.Println("TEST PASSED: " + test_name)
}

/*************
* TEST CASES *
**************/
func (test Test) TestRequestWithoutSQLI() bool {
  return test.Test("/",
           Header{
             "header-key-0": "header-value-0",
	   },
	   Cookie{
	     "cookie-name-0": "cookie-value-0",
	   },
	   Body{
             "body-key-0": "body-value-0",
	   },
           false)
}

func (test Test) TestSQLIinPath() {
}

func TestSQLIinHeader() {
}

func TestSQLIinCookie() {
}

func TestSQLIinBody() {
}

func TestSQLIExcludedByConfig() {
}

func main() {
  // begin testing
  fmt.Println("====== Integration test starting ======")
  request_received := make(chan *http.Request)
  body_string_received := make(chan string)

  // set up test suite
  test := Test{
    proxy_url: "http://proxy:8000",
    server_port: ":8080",
    server_message: "request received",
    request_received: request_received,
    body_string_received: body_string_received,
  }
  test.StartServer()
  test.CheckProxyConnection()

  // start testing
  run("Test Request Without SQLI", test.TestRequestWithoutSQLI())

  // terminate with sucess
  fmt.Println("====== Passed all integration tests ======")
}

