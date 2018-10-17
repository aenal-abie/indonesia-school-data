package main

import (
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/fetchbot"
	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/spf13/viper"
)

var (
	db               *gorm.DB
	mu               sync.Mutex
	resourceUrl      string
	connectionString string
	err              error
)

// Duplicates table
var dup = map[string]bool{}

type Village struct {
	ID          int
	District_id int
	Name        string
}

type School struct {
	ID                int    `json: "id"`
	Name              string `json: "name"`
	Village_name      string `json: "village_name"`
	District_id       int    `json: "district_id"`
	Address           string `json: "address"`
	Postal_code       string `json: "postal_code"`
	Status            string
	Educational_level string `json: "educational_level"`
}

var masterPattern = `[\w .,\(\)\x60\'-]+`

func loadConfig() {

	viper.AddConfigPath(".")

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

}

func openDatabase() {
	db, err = gorm.Open("mysql", connectionString)

	if err != nil {
		panic("Cannot connect to database: " + err.Error())
	}

}

func main() {

	loadConfig()

	updateProvince := flag.Bool("update-provinces", false, "refetch provinces data")
	updateRegency := flag.Bool("update-regencies", false, "refetch regencies data")
	updateDistrict := flag.Bool("update-districts", false, "refetch disctricts data")
	startFromDistrict := flag.String("start-from-district", "", "start from district id to recrawl")

	flag.Parse()

	connectionString = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		viper.Get("DB_USER"),
		viper.Get("DB_PASSWORD"),
		viper.Get("DB_HOST"),
		viper.Get("DB_PORT"),
		viper.Get("DB_DATABASE"),
	)

	openDatabase()

	// db, err := gorm.Open("mysql", connectionString)

	// if err != nil {
	// 	panic("Cannot connect to database: " + err.Error())
	// }

	// defer db.Close()

	db.AutoMigrate(&School{})

	if *updateProvince {
		updateProvinces()
	}

	if *updateRegency {
		updateRegencies()
	}

	if *updateDistrict {
		updateDistricts()
	}

	resourceUrl = viper.GetString("RESOURCE_URL")

	var districts []District

	// db, err = gorm.Open("mysql", connectionString)

	// defer db.Close()

	// if err != nil {
	// 	panic("Cannot connect to database: " + err.Error())
	// }
	// fmt.Printf("%#v", *startFromDistrict)
	if *startFromDistrict != "" {
		db.Where("code >= ?", *startFromDistrict).Find(&districts)
	} else {
		db.Find(&districts)
	}

	f := fetchbot.New(fetchbot.HandlerFunc(handler))

	queue := f.Start()

	for _, district := range districts {
		url := fmt.Sprintf(resourceUrl, district.Code)
		// url := "http://referensi.data.kemdikbud.go.id/index11.php?kode=010101&level=3"

		// queue.SendStringGet(url)
		// resourceUrl = "http://referensi.data.kemdikbud.go.id/index11.php?kode=010101&level=3"
		_, err = queue.SendStringGet(url)
		// _, err = queue.SendStringGet(url + "&id=15")
		// _, err = queue.SendStringGet(url + "&id=16")
		// _, err = queue.SendStringGet(url + "&id=37")
		// _, err = queue.SendStringGet(url + "&id=39")

		if err != nil {
			fmt.Printf("[ERR] GET %s - %s\n", url, err)
		}
	}

	queue.Close()
}

func handler(ctx *fetchbot.Context, res *http.Response, err error) {
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

	queueSingleSchoolData(ctx, doc)

}

func queueSingleSchoolData(ctx *fetchbot.Context, doc *goquery.Document) {
	mu.Lock()

	// newF := fetchbot.New(fetchbot.HandlerFunc(getSchollDataHandler))
	// newQueue := newF.Start()
	kode := parseKode(ctx.Cmd.URL().RequestURI())
	district := District{}

	db.Where("code = ?", kode).First(&district)

	doc.Find("table td a[title=\"Tampilkan Profil Sekolah\"]").Each(func(i int, s *goquery.Selection) {
		val, _ := s.Attr("href")
		school := School{}

		// val = "http://referensi.data.kemdikbud.go.id/tabs.php?npsn=20231720"
		npsn := parseNPSNFromUrl(val)
		// os.Exit(1)

		db.Model(&school).Where("id = ?", npsn).Update("district_id", district.ID)
		fmt.Printf("[UPDATE] %s %s\n", npsn, kode)

		// Resolve address
		// u, err := ctx.Cmd.URL().Parse(val)
		// if err != nil {
		// 	fmt.Printf("error: resolve URL %s - %s\n", val, err)
		// 	return
		// }
		// if !dup[u.String()] {
		// 	if _, err := newQueue.SendStringGet(u.String()); err != nil {
		// 		fmt.Printf("error: enqueue get %s - %s\n", u, err)
		// 	} else {
		// 		dup[u.String()] = true
		// 	}
		// }
	})

	// newQueue.Close()

	mu.Unlock()
}

func getSchollDataHandler(ctx *fetchbot.Context, res *http.Response, err error) {
	doc, err := goquery.NewDocumentFromResponse(res)

	if err != nil {
		fmt.Printf("[ERR] %s %s - %s\n", ctx.Cmd.Method(), ctx.Cmd.URL(), err)
		return
	}

	parseSingleSchoolData(doc)
}

func parseSingleSchoolData(doc *goquery.Document) {

	var district District

	// openDatabase()
	// db, err := gorm.Open("mysql", connectionString)

	// if err != nil {
	// 	panic("Cannot connect to database: " + err.Error())
	// }
	// defer db.Close()

	tableContent := doc.Find("td[width=\"70%\"] table").First().Text()

	// cleanup data
	tableContent = strings.Replace(tableContent, "\t", "", -1)
	tableContent = strings.Replace(tableContent, "\n\n", "", -1)
	tableContent = strings.Replace(tableContent, "\u00a0", "", -1)

	school := School{}
	npsn, _ := strconv.Atoi(parseNPSN(tableContent))
	school.ID = npsn
	school.Name = parseName(tableContent)
	school.Address = parseAddress(tableContent)
	school.Postal_code = parsePostalCode(tableContent)
	school.Status = parseStatus(tableContent)
	school.Educational_level = parseEducationalLevel(tableContent)
	school.Village_name = parseVillageName(tableContent)

	districtName := parseDistrictName(tableContent)
	districtName = strings.Replace(districtName, "Kec. ", "", -1)

	if err := db.Where("name = ?", "Kec. "+districtName).First(&district).Error; err != nil {
		fmt.Printf("Error query: ", err.Error())
	}
	fmt.Printf("%#v", districtName, district.ID)

	school.District_id = district.ID

	db.Where(School{ID: school.ID}).Assign(school).FirstOrCreate(&school)
	fmt.Printf("[SCHOOL] ID %d parsed\n", school.ID)
}

func parseKode(s string) string {
	r := regexp.MustCompile(`kode=(\d+)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return match[1]
	}

	return ""
}

func parseName(s string) string {
	r := regexp.MustCompile(`Nama\n:\n(` + masterPattern + `)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseNPSNFromUrl(s string) string {
	r := regexp.MustCompile(`npsn=(\d+)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseNPSN(s string) string {
	r := regexp.MustCompile(`NPSN\n:\n(` + masterPattern + `)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseAddress(s string) string {
	r := regexp.MustCompile(`Alamat\n:\n(` + masterPattern + `)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parsePostalCode(s string) string {
	r := regexp.MustCompile(`Kode Pos\n:\n(\w+)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseStatus(s string) string {
	r := regexp.MustCompile(`Status Sekolah\n:\n(\w+)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseEducationalLevel(s string) string {
	r := regexp.MustCompile(`Jenjang Pendidikan\n:\n(\w+)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseVillageName(s string) string {
	r := regexp.MustCompile(`Desa/Kelurahan\n:\n(` + masterPattern + `)`)

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}

func parseDistrictName(s string) string {
	r := regexp.MustCompile(`Kecamatan\/Kota \(LN\)\n:\n(` + masterPattern + `)`) // \x60 support for backtics(`) character

	match := r.FindStringSubmatch(s)

	if len(match) > 1 {
		return strings.Trim(match[1], " ")
	}

	return ""
}
