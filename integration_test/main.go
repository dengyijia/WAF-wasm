package main

import (
	"fmt"
)

// Config struct reflects the expectations for the WASM filter configuration
type Config struct {
	configName string
	proxyURL   string

	// the headers that are expected to be included/excluded in detection
	headerInclude []string
	headerExclude []string

	// true if the config for cookie contains `include`
	// in this case, only the cookies specified in cookieIncluded will
	// be included in detection
	cookieIncludeBool bool

	// the cookie names that are expected to be included/excluded in detection
	cookieInclude []string
	cookieExclude []string

	// true if the config for body contains `include`
	bodyIncludeBool bool

	// the body param keys that are expected to be included/excluded in detection
	bodyInclude []string
	bodyExclude []string
}

func TestForConfig(config Config) {
	suite := NewTestSuite(config.proxyURL)
	sqliString := "=1' AND 1=1"
	var request TestRequest

	// no SQL injection
	suite.requests["Request without SQL injection"] = NewTestRequest()

	// path
	request = NewTestRequest()
	request.pathParams.Set("path-key-0", sqliString)
	request.sqli = true
	suite.requests["Request with SQL injection in path value"] = request

	request = NewTestRequest()
	request.pathParams.Set(sqliString, "path-val-0")
	request.sqli = true
	suite.requests["Request with SQL injection in path key"] = request

	// header
	for _, key := range config.headerInclude {
		request = NewTestRequest()
		request.headers[key] = sqliString
		request.sqli = true
		suite.requests["Request with SQL injection in included header: "+key] = request
	}
	for _, key := range config.headerExclude {
		request = NewTestRequest()
		request.headers[key] = sqliString
		request.sqli = false
		suite.requests["Request with SQL injection in excluded header: "+key] = request
	}

	// cookie
	for _, key := range config.cookieInclude {
		request = NewTestRequest()
		request.cookies[key] = sqliString
		request.sqli = true
		suite.requests["Request with SQL injection in included cookie: "+key] = request
	}
	for _, key := range config.cookieExclude {
		request = NewTestRequest()
		request.cookies[key] = sqliString
		request.sqli = false
		suite.requests["Request with SQL injection in excluded cookie: "+key] = request
	}
	if !config.cookieIncludeBool {
		request = NewTestRequest()
		request.cookies[sqliString] = "value"
		request.sqli = true
		suite.requests["Request with SQL injection in cookie key"] = request
	}

	// body
	for _, key := range config.bodyInclude {
		request = NewTestRequest()
		request.body.Set(key, sqliString)
		request.sqli = true
		request.sqli_body = true
		suite.requests["Request with SQL injection in included body param: "+key] = request
	}
	for _, key := range config.bodyExclude {
		request = NewTestRequest()
		request.body.Set(key, sqliString)
		request.sqli = false
		suite.requests["Request with SQL injection in excluded body param: "+key] = request
	}
	if !config.bodyIncludeBool {
		request = NewTestRequest()
		request.body.Set(sqliString, "value")
		request.sqli = true
		request.sqli_body = true
		suite.requests["Request with SQL injection in body key"] = request
	}
	fmt.Println("------ Tests for ", config.configName, "Config starts ------")
	suite.RunRequests()
	fmt.Println("------ All tests for ", config.configName, "Config passed ------")
}

func TestDefault() {
	// This test corresponds to the configuration in envoy-default.yaml
	c := Config{
		configName:        "Default",
		proxyURL:          "http://proxy-default:8081",
		headerInclude:     []string{"referer", "user-agent"},
		headerExclude:     []string{"header-key-1"},
		cookieIncludeBool: false,
		cookieInclude:     []string{"cookie-key-1"},
		cookieExclude:     []string{},
		bodyIncludeBool:   false,
		bodyInclude:       []string{"body-key-1"},
		bodyExclude:       []string{},
	}
	TestForConfig(c)
}

func TestInclude() {
	// This test corresponds to the configuration in envoy-include.yaml
	c := Config{
		configName:        "Include",
		proxyURL:          "http://proxy-include:8082",
		headerInclude:     []string{"header-key-0", "referer", "user-agent"},
		headerExclude:     []string{"header-key-1"},
		cookieIncludeBool: true,
		cookieInclude:     []string{"cookie-key-0"},
		cookieExclude:     []string{"cookie-key-1"},
		bodyIncludeBool:   true,
		bodyInclude:       []string{"body-key-0"},
		bodyExclude:       []string{"body-key-1"},
	}
	TestForConfig(c)
}

func TestExclude() {
	// This test corresponds to the configuration in envoy-exclude.yaml
	c := Config{
		configName:        "Exclude",
		proxyURL:          "http://proxy-exclude:8083",
		headerInclude:     []string{"user-agent"},
		headerExclude:     []string{"header-key-0", "referer"},
		cookieIncludeBool: false,
		cookieInclude:     []string{"cookie-key-1"},
		cookieExclude:     []string{"cookie-key-0"},
		bodyIncludeBool:   false,
		bodyInclude:       []string{"body-key-1"},
		bodyExclude:       []string{"body-key-0"},
	}
	TestForConfig(c)
}
func main() {
	fmt.Println("====== Integration tests start ======")
	TestDefault()
	TestInclude()
	TestExclude()
	fmt.Println("====== All integration tests passed ======")
}
