package main
import (
	"fmt"
)

type Config struct {
	config_name string
	proxy_url string
	header_include_bool bool
	header_include []string
	header_exclude []string
	cookie_include_bool bool
	cookie_include []string
	cookie_exclude []string
	body_include_bool bool
	body_include []string
	body_exclude []string
}

func TestForConfig(config Config) {
	suite := NewTestSuite(config.proxy_url)
	sqli_string := "=1' AND 1=1"
	var request TestRequest

	// no SQL injection
	suite.requests["Request without SQL injection"] = NewTestRequest()

	// path
	request = NewTestRequest()
	request.pathParams.Set("path-key-0", sqli_string)
	request.sqli = true
	suite.requests["Request with SQL injection in path value"] = request

	request = NewTestRequest()
	request.pathParams.Set(sqli_string, "path-val-0")
	request.sqli = true
	suite.requests["Request with SQL injection in path key"] = request

	// header
	for _, key := range config.header_include {
		request = NewTestRequest()
		request.headers[key] = sqli_string
		request.sqli = true
		suite.requests["Request with SQL injection in included header: " + key] = request
	}
	for _, key := range config.header_exclude {
		request = NewTestRequest()
		request.headers[key] = sqli_string
		request.sqli = false
		suite.requests["Request with SQL injection in excluded header: " + key] = request
	}

	// cookie
	for _, key := range config.cookie_include {
		request = NewTestRequest()
		request.cookies[key] = sqli_string
		request.sqli = true
		suite.requests["Request with SQL injection in included cookie: " + key] = request
	}
	for _, key := range config.cookie_exclude {
		request = NewTestRequest()
		request.cookies[key] = sqli_string
		request.sqli = false
		suite.requests["Request with SQL injection in excluded cookie: " + key] = request
	}
	if config.cookie_include_bool == false {
		request = NewTestRequest()
		request.cookies[sqli_string] = "value"
		request.sqli = true
		suite.requests["Request with SQL injection in cookie key"] = request
	}

	// body
	/*
	for _, key := range config.body_include {
		request = NewTestRequest()
		request.body.Add(key, sqli_string)
		request.sqli = true
		suite.requests["Request with SQL injection in included body param: " + key] = request
	}*/
	for _, key := range config.body_exclude {
		request = NewTestRequest()
		request.body.Add(key, sqli_string)
		request.sqli = false
		suite.requests["Request with SQL injection in excluded body param: " + key] = request
	}
	/*
	if config.body_include_bool == false {
		request = NewTestRequest()
		request.body.Add(sqli_string, "value")
		request.sqli = true
		suite.requests["Request with SQL injection in body key"] = request
	}*/
	fmt.Println("------ Tests for ", config.config_name, "Config starts ------")
	suite.RunRequests()
	fmt.Println("------ All tests for ", config.config_name, "Config passed ------")
}

func TestDefault() {
	c := Config{
		config_name: "Default",
		proxy_url: "http://proxy-default:8081",
		header_include_bool: true,
		header_include: []string{"referer", "user-agent"},
		header_exclude: []string{"header-key-1"},
		cookie_include_bool: false,
		cookie_include: []string{"cookie-key-1"},
		cookie_exclude: []string{},
		body_include_bool: false,
		body_include: []string{"body-key-1"},
		body_exclude: []string{},
	}
	TestForConfig(c)
}

func TestInclude() {
	c := Config{
		config_name: "Include",
		proxy_url: "http://proxy-include:8082",
		header_include_bool: true,
		header_include: []string{"header-key-0", "referer", "user-agent"},
		header_exclude: []string{"header-key-1"},
		cookie_include_bool: true,
		cookie_include: []string{"cookie-key-0"},
		cookie_exclude: []string{"cookie-key-1"},
		body_include_bool: true,
		body_include: []string{"body-key-0"},
		body_exclude: []string{"body-key-1"},
	}
	TestForConfig(c)
}

func TestExclude() {
	c := Config{
		config_name: "Exclude",
		proxy_url: "http://proxy-exclude:8083",
		header_include_bool: false,
		header_include: []string{"user-agent"},
		header_exclude: []string{"header-key-0", "referer"},
		cookie_include_bool: false,
		cookie_include: []string{"cookie-key-1"},
		cookie_exclude: []string{"cookie-key-0"},
		body_include_bool: false,
		body_include: []string{"body-key-1"},
		body_exclude: []string{"body-key-0"},
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
