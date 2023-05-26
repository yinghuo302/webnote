package main

import (
	"os"
	"strconv"
	"webnote/config"
	"webnote/db"
	"webnote/server"

	"github.com/gin-gonic/gin"
)

func main() {
	engine := gin.Default()
	config.Init()
	server.Init(engine)
	db.InitDB()
	path, _ := os.Getwd()
	println(path)
	engine.Run(config.Conf.Address + ":" + strconv.Itoa(config.Conf.Port))
}
