package main

func TestDefault() {

	suite := NewTestSuite("http://proxy-default:8000")
	suite.StartServer()
	suite.CheckProxyConnection()
	suite.RunSuite()
	suite.CloseServer()
}


func main() {
	

}
