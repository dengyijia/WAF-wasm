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

/*
* type aliases
*/
type Header = map[string]string
type Cookie = map[string]string
type Body = map[string]string

type Test struct {
  proxy_url string
  server_message string
  request_received chan *http.Request
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
  request_body := strings.NewReader(request_body_url.Encode())
  request, _ := http.NewRequest("POST", t.proxy_url + path, request_body)
  for key, val := range headers {
    request.Header.Set(key, val)
  }
  for key, val := range cookies {
    request.AddCookie(&http.Cookie{Name: key, Value: val})
  }

  // send request
  client := &http.Client{}
  response, err := client.Do(request)
  if err != nil {
    fmt.Println("Send request failed")
    return false
  }
  defer response.Body.Close()

  // check results of the request
  var received *http.Request
  select {
    case received = <-t.request_received:
    default: received = nil
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
	return false
      }
    }
    for _, cookie_received := range received.Cookies() {
      cookie_sent, found := cookies[cookie_received.Name]
      if found == false || cookie_sent != cookie_received.Value {
	fmt.Println("Request without SQL injection has cookie altered by proxy")
	return false
      }
    }
    if received.Body != request.Body {
      fmt.Println("Request without SQL injection has body altered by proxy")
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

func main() {
  // begin testing
  fmt.Println("====== Integration test starting ======")
  proxy_url := "http://proxy:8000"
  server_port := ":8080"
  server_message := "request received"

  // start the server
  request_received := make(chan *http.Request)
  http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, server_message)
    go func(){ request_received <- r }()
  })
  go http.ListenAndServe(server_port, nil)
  fmt.Println("Server started")

  // wait and test that the proxy is up
  time.Sleep(2 * time.Second)
  response, err := http.Get(proxy_url)
  for err != nil {
    fmt.Println("Client connecting to proxy ...")
    time.Sleep(2 * time.Second)
    response, err = http.Get(proxy_url)
  }
  defer response.Body.Close()

  // check that proxy reply comes from server
  response_body, _ := ioutil.ReadAll(response.Body)
  if response.StatusCode != 200 || string(response_body) != server_message {
    fmt.Println("Proxy connection failed")
  }
  fmt.Println("Client connected to server through proxy")

  // set up test suite
  test := Test{
    proxy_url: proxy_url,
    server_message: server_message,
    request_received: request_received,
  }

  // start testing
  if !test.Test("/something",
           Header{
             "header-key-0": "header-value-0",
	   },
	   Cookie{
	     "cookie-name-0": "cookie-value-0",
	   },
	   Body{
             "body-key-0": "body-value-0",
	   },
           false) {
    fail()
  }
  fmt.Println("TEST passed")

  // terminate with sucess
  fmt.Println("====== Passed all integration tests ======")
}

