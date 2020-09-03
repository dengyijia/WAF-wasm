package main

import (
	"fmt"
)

// A Config struct object reflects the expectations for one WASM filter configuration
// running at a given URL
// Specific rules for configuration can be found in the README.md at root of the repository
type Config struct {
	configName string
	// the proxy with the filter that corresponds to the configs expected in this struct object
	// is listening at this url.
	// all test requests for this config will be made to this url.
	proxyURL   string

	// the headers that are expected to be included/excluded in detection
	wantHeaderInclude []string
	wantHeaderExclude []string

	// true if cookies with unexpected names are expected to go through detection
	wantCookieIncludeUnexpected bool

	// the cookie names that are expected to be included/excluded in detection
	wantCookieInclude []string
	wantCookieExclude []string

	// true if the body parameters with unexpected keys are expected to go through detection
	wantBodyIncludeUnexpected bool

	// the body param keys that are expected to be included/excluded in detection
	wantBodyInclude []string
	wantBodyExclude []string
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
	request.containsSQLi = true
	suite.requests["Request with SQL injection in path value"] = request

	request = NewTestRequest()
	request.pathParams.Set(sqliString, "path-val-0")
	request.containsSQLi = true
	suite.requests["Request with SQL injection in path key"] = request

	// header
	for _, key := range config.wantHeaderInclude {
		request = NewTestRequest()
		request.headers[key] = sqliString
		request.containsSQLi = true
		suite.requests["Request with SQL injection in included header: "+key] = request
	}
	for _, key := range config.wantHeaderExclude {
		request = NewTestRequest()
		request.headers[key] = sqliString
		request.containsSQLi = false
		suite.requests["Request with SQL injection in excluded header: "+key] = request
	}

	// cookie
	for _, key := range config.wantCookieInclude {
		request = NewTestRequest()
		request.cookies[key] = sqliString
		request.containsSQLi = true
		suite.requests["Request with SQL injection in included cookie: "+key] = request
	}
	for _, key := range config.wantCookieExclude {
		request = NewTestRequest()
		request.cookies[key] = sqliString
		request.containsSQLi = false
		suite.requests["Request with SQL injection in excluded cookie: "+key] = request
	}
	if config.wantCookieIncludeUnexpected {
		request = NewTestRequest()
		request.cookies[sqliString] = "value"
		request.containsSQLi = true
		suite.requests["Request with SQL injection in cookie key"] = request
	}

	// body
	for _, key := range config.wantBodyInclude {
		request = NewTestRequest()
		request.body.Set(key, sqliString)
		request.containsSQLi = true
		request.containsSQLiInBody = true
		suite.requests["Request with SQL injection in included body param: "+key] = request
	}
	for _, key := range config.wantBodyExclude {
		request = NewTestRequest()
		request.body.Set(key, sqliString)
		request.containsSQLi = false
		suite.requests["Request with SQL injection in excluded body param: "+key] = request
	}
	if config.wantBodyIncludeUnexpected {
		request = NewTestRequest()
		request.body.Set(sqliString, "value")
		request.containsSQLi = true
		request.containsSQLiInBody = true
		suite.requests["Request with SQL injection in body key"] = request
	}
	fmt.Println("------ Tests for ", config.configName, "Config starts ------")
	suite.RunRequests()
	fmt.Println("------ All tests for ", config.configName, "Config passed ------")
}

func TestDefault() {
	/* This test corresponds to the configuration in integration_test/envoy-default.yaml
	* No configuration string is passed and the default configuration will apply
	* Two headers "referer" and "user-agent", all cookies, and all body parameters are
	* expected to go through SQL injection detection
	*/
	c := Config{
		configName:        "Default",
		proxyURL:          "http://proxy-default:8081",
		wantHeaderInclude:     []string{"referer", "user-agent"},
		wantHeaderExclude:     []string{"header-key-1"},
		wantCookieIncludeUnexpected: false,
		wantCookieInclude:     []string{"cookie-key-1"},
		wantCookieExclude:     []string{},
		wantBodyIncludeUnexpected:   false,
		wantBodyInclude:       []string{"body-key-1"},
		wantBodyExclude:       []string{},
	}
	TestForConfig(c)
}

func TestInclude() {
	/* This test corresponds to the configuration in integration_test/envoy-include.yaml
	{
		"body": {
			"content-type": "application/x-www-form-urlencoded",
			"include": ["body-key-0"]
		},
		"header": {
			"include": ["header-key-0"]
		},
		"cookie": {
			"include": ["cookie-key-0"]
		}
	}
	Only the body, header, and cookie fields explicitly specified are expected to go through
	SQL injection detection.
	*/
	c := Config{
		configName:        "Include",
		proxyURL:          "http://proxy-include:8082",
		wantHeaderInclude:     []string{"header-key-0"},
		wantHeaderExclude:     []string{"header-key-1"},
		wantCookieIncludeUnexpected: false,
		wantCookieInclude:     []string{"cookie-key-0"},
		wantCookieExclude:     []string{"cookie-key-1"},
		wantBodyIncludeUnexpected:   false,
		wantBodyInclude:       []string{"body-key-0"},
		wantBodyExclude:       []string{"body-key-1"},
	}
	TestForConfig(c)
}

func TestExclude() {
	/* This test corresponds to the configuration in envoy-exclude.yaml
	The following config is applied:
	{
		"body": {
			"content-type": "application/x-www-form-urlencoded",
			"exclude": ["body-key-0"]
		},
		"header": {
			"exclude": ["header-key-0"]
		},
		"cookie": {
			"exclude": ["cookie-key-0"]
		}
	}
	All but the body, header, and cookie fields explicitly specified are expected to go through
	SQL injection detection.
	*/
	c := Config{
		configName:        "Exclude",
		proxyURL:          "http://proxy-exclude:8083",
		wantHeaderInclude:     []string{"header-key-1"},
		wantHeaderExclude:     []string{"header-key-0"},
		wantCookieIncludeUnexpected: true,
		wantCookieInclude:     []string{"cookie-key-1"},
		wantCookieExclude:     []string{"cookie-key-0"},
		wantBodyIncludeUnexpected:   true,
		wantBodyInclude:       []string{"body-key-1"},
		wantBodyExclude:       []string{"body-key-0"},
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
