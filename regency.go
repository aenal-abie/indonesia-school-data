package main

import (
	"fmt"
	"net/http"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
)

type Regency struct {
	ID          int       `json: "id"`
	Code        string    `json: "code"`
	Province_id int       `json: "province_id"`
	Province    *Province `gorm:"ForeignKey:Province_id"`
	Name        string
}

func updateRegencies() {

	var provinces []Province

	db, err := gorm.Open("mysql", connectionString)

	defer db.Close()

	db.Find(&provinces)

	if err != nil {
		panic("Cannot connect to database: " + err.Error())
	}

	f := fetchbot.New(fetchbot.HandlerFunc(regencyHandler))
	queue := f.Start()

	resourceUrl := "http://referensi.data.kemdikbud.go.id/index11.php?kode=%s&level=1"

	for _, province := range provinces {
		url := fmt.Sprintf(resourceUrl, province.Code)
		_, err := queue.SendStringGet(url)
		if err != nil {
			fmt.Printf("[ERR] GET %s - %s\n", url, err)
		}
	}

	queue.Close()

}

func regencyHandler(ctx *fetchbot.Context, res *http.Response, err error) {
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

	saveRegencies(ctx, doc)
}

func saveRegencies(ctx *fetchbot.Context, doc *goquery.Document) {
	var regency Regency
	var province Province

	db, err := gorm.Open("mysql", connectionString)
	defer db.Close()

	if err != nil {
		panic("Cannot connect to database: " + err.Error())
	}
	db.AutoMigrate(&Regency{})

	doc.Find("#box-table-a tbody td a").Each(func(i int, s *goquery.Selection) {
		link, _ := s.Attr("href")

		code := parseKode(link)

		if code != "" {
			regency = Regency{}
			regency.Code = code
			regency.Name = s.Text()

			provinceCode := string(code[0:2]) + "0000"

			if err := db.Where("code = ?", provinceCode).First(&province).Error; err != nil {
				fmt.Printf("Error query: %s", err.Error())

			} else {
				regency.Province_id = province.ID
			}

			db.Create(&regency)
		}

	})
}
