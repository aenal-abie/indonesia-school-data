package main

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
)

type Province struct {
	ID   int
	Code string
	Name string
}

func updateProvinces() {

	f := fetchbot.New(fetchbot.HandlerFunc(provinceHandler))
	queue := f.Start()
	url := "http://referensi.data.kemdikbud.go.id/index11.php"

	_, err := queue.SendStringGet(url)
	if err != nil {
		fmt.Printf("[ERR] GET %s - %s\n", url, err)
	}

	queue.Close()

}

func provinceHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err)
		return
	}
	fmt.Printf("[%d] %s %s\n", res.StatusCode, ctx.Cmd.Method(), ctx.Cmd.URL())

	doc, err := goquery.NewDocumentFromResponse(res)

	if err != nil {
		fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
		return
	}

	saveProvinces(ctx, doc)
}

func saveProvinces(ctx *fetchbot.Context, doc *goquery.Document) {
	var province Province

	db, err := gorm.Open("mysql", connectionString)
	defer db.Close()

	if err != nil {
		panic("Cannot connect to database: " + err.Error())
	}
	db.AutoMigrate(&Province{})

	doc.Find("#box-table-a tbody td a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")

		code := parseKode(link)

		if code != "" {
			province = Province{}
			province.Code = code
			province.Name = s.Text()

			db.Create(&province)
		}

	})
}
