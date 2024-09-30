package main

import (
	"github.com/csr-ugra/avito-estate-parser/src/util"
	"github.com/go-rod/rod"
	"time"
)

func main() {
	config := util.GetConfig()

	util.InitLogger(config)

	//connection, err := db.GetConnection(config)
	//if err != nil {
	//	log.Fatalln(err)
	//}

	//ctx := context.Background()

	page := rod.New().ControlURL(config.DevtoolsWebsocketUrl.Value).MustConnect().MustPage("https://example.com")
	defer page.MustClose()
	time.Sleep(30 * time.Minute)
	_ = page

}
