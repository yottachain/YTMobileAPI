package router

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/yottachain/YTMobileAPI/httpServer"
)

func InitRouter() (router *gin.Engine) {
	router = gin.Default()
	gin.SetMode(gin.DebugMode)
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	router.Use(cors.New(config))

	v1 := router.Group("/api/v1")
	{
		v1.GET("/downloadObject", httpServer.DownloadObject)
		v1.GET("/getRandomKey", httpServer.GetPubKey)
		v1.GET("addUserToS3", httpServer.AddUserToS3server)
	}

	return
}
