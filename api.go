package main
import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var priceRegex = regexp.MustCompile("^\\d+\\.*\\d*$")

type Item struct {
	ShortDescription, Price string
}

type Receipt struct {
	Retailer, PurchaseDate, PurchaseTime, Total string
	Items []Item
}

type Record struct {
	receipt Receipt
	id string
	points int64
}

var m map[string]int64

func main() {
	m = make(map[string]int64)
	router := chi.NewRouter()
	router.Use(middleware.Logger)
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World!"))
	})
	router.Route("/receipts", func(r chi.Router) {
		r.Get("/{id}/points", getReceiptPoints)                                          // GET /articles/123
		r.Post("/process", processReceipt)                                       // PUT /articles/123                                // DELETE /articles/123
	})
	http.ListenAndServe(":3000", router)
}

func getReceiptPoints(w http.ResponseWriter, r *http.Request) {
	points := m[chi.URLParam(r, "id")]
	fmt.Fprintf(w, "{\"points\": %d}", points)
}

func processReceipt(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var receipt Receipt
	err := decoder.Decode(&receipt)
	if err != nil {
		panic(err)
	}
	
	id := calculatePoints(&receipt)
	fmt.Fprintf(w, "{\"id\": %s}", id)
}

func calculatePoints(receipt *Receipt) string {
	var points int64
	points = 0
	
	//add points for retailer alphanumeric count
	regex := regexp.MustCompile("[[:alnum:]]")
	points += int64(len(regex.FindAllString (receipt.Retailer, -1)))
	
	//add points for total round dollar amount & multiple of .25
	if priceRegex.MatchString(receipt.Total) {
		total, _ := decimal.NewFromString(receipt.Total)
		if total.IsInteger() {
			points += 50
		}
		tf, _ := decimal.NewFromString(".25")
		if total.Mod(tf).Equal(decimal.Zero) {
			points += 25
		}
	}
	
	//add 5 poinst for every 2 items
	points += int64((len(receipt.Items) / 2) * 5)
	
	//add item price times .2 rounded up to points if item description is evenly divisble by 3
	for _, item := range receipt.Items {
		if len(strings.TrimSpace(item.ShortDescription)) % 3 == 0 && priceRegex.MatchString(item.Price) {
			price, _ := decimal.NewFromString(item.Price)
			t, _ := decimal.NewFromString(".2")
			points += price.Mul(t).RoundUp(0).IntPart()
		}
	}
	
	//add points for odd date
	dateArray := strings.Split(receipt.PurchaseDate, "-")
	if len(dateArray) >= 3{
		day, dayError := strconv.Atoi(dateArray[2])
		if dayError == nil && day % 2 != 0 {
			points += 6
		}
	}
	
	//add points for after 2p & before 4p inclusive
	hour, hourError := strconv.Atoi(strings.Split(receipt.PurchaseTime, ":")[0])
	if hourError == nil && 14 <= hour  && hour <= 16 {
		points += 10
	}
	
	id := uuid.NewString()
	m[id] = points
	return id
}