package main

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
)

type District struct {
	ID         int    `json: "id"`
	Regency_id int    `json: "regency_id"`
	Code       string `json: "code"`
	Name       string `json: "name"`
}

func updateDistricts() {

	var regencies []Regency

	db, err := gorm.Open("mysql", connectionString)

	defer db.Close()

	db.Find(&regencies)

	if err != nil {
		panic("Cannot connect to database: " + err.Error())
	}

	f := fetchbot.New(fetchbot.HandlerFunc(districtHandler))
	queue := f.Start()

	resourceUrl := "http://referensi.data.kemdikbud.go.id/index11.php?kode=%s&level=2"

	for _, regency := range regencies {
		url := fmt.Sprintf(resourceUrl, regency.Code)
		_, err := queue.SendStringGet(url)
		if err != nil {
			fmt.Printf("[ERR] GET %s - %s\n", url, err)
		}
	}

	queue.Close()

}

func districtHandler(ctx *fetchbot.Context, res *http.Response, err error) {
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

	saveDiscticts(ctx, doc)
}

func saveDiscticts(ctx *fetchbot.Context, doc *goquery.Document) {
	var district District
	var regency Regency

	db, err := gorm.Open("mysql", connectionString)
	defer db.Close()

	if err != nil {
		panic("Cannot connect to database: " + err.Error())
	}
	db.AutoMigrate(&District{})

	doc.Find("#box-table-a tbody td a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")

		code := parseKode(link)

		if code != "" {
			district = District{}
			district.Code = code
			district.Name = s.Text()

			regencyCode := string(code[0:4]) + "00"

			if err := db.Where("code = ?", regencyCode).First(&regency).Error; err != nil {
				fmt.Printf("Error query: %s", err.Error())

			} else {
				district.Regency_id = regency.ID
			}

			db.Create(&district)
		}

	})
}
