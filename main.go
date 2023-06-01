package main

import (
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/xid"
)

// host and port the app is running on
const HOST = "127.0.0.1"
const PORT = ":8080"

// values for calculating how many points a receipt is worth
const VALUE_PER_ALPHANUMERIC_CHAR = 1
const VALUE_PER_TWO_ITEMS = 5
const ROUND_DOLLAR_AMOUNT_BONUS = 50
const MULTIPLE_OF_0_POINT_25_BONUS = 25
const ODD_DAY_BONUS = 6
const BETWEEN_2PM_AND_4PM_BONUS = 10
const ITEM_PRICE_MULTIPLIER = 0.2

// response for aborted endpoints, the description of the error
type Description struct {
	Description string `json:"description"`
}

// response of /receipts/process endpoint, the id of the new receipt
type Id struct {
	Id string `json:"id"`
}

// response of /receipts/:id/points endpoint, the number of points awarded to the given receipt
type Points struct {
	Points int `json:"points"`
}

// a specific item purchased
type Item struct {
	ShortDescription string `json:"shortDescription" binding:"required"`
	Price            string `json:"price" binding:"required"`
}

// a receipt
type Receipt struct {
	Retailer     string  `json:"retailer" binding:"required"`
	PurchaseDate string  `json:"purchaseDate" binding:"required"`
	PurchaseTime string  `json:"purchaseTime" binding:"required"`
	Total        string  `json:"total" binding:"required"`
	Items        []*Item `json:"items" binding:"required"`
}

// map of all receipts processed, a real implementation would use a database
var receipts map[string]Receipt = make(map[string]Receipt)

func main() {
	router := gin.Default()
	router.POST(`/receipts/process`, processReceipts)
	router.GET(`/receipts/:id/points`, getPoints)

	router.Run(HOST + PORT)
}

/*
Processes the given receipt and adds it to the receipts map
responds with the unique id assigned to the receipt
*/
func processReceipts(context *gin.Context) {
	var receipt Receipt

	// attempt to create a Receipt struct from the given JSON object, abort on failure with 400 error
	err := context.ShouldBindJSON(&receipt)
	if err != nil {
		context.AbortWithStatusJSON(http.StatusBadRequest, Description{Description: "The receipt is invalid"})
		return
	}

	// use xid to create a random, unique id for the receipt and add it to the receipts map
	id := xid.New().String()
	receipts[id] = receipt

	// return the id as a json object with a 200 status
	context.JSON(http.StatusOK, Id{Id: id})
}

/*
Calculates the number of points a given receipt is worth
takes the id of the receipt via url param
responds with the number of points the receipt is worth
*/
func getPoints(context *gin.Context) {
	// the id comes from the url
	id := context.Param("id")

	// attempt to find the receipt from the receipts map, abort on failure with 404 error
	receipt, found := receipts[id]
	if !found {
		context.AbortWithStatusJSON(http.StatusBadRequest, Description{Description: "No receipt found for that id"})
		return
	}

	points := 0

	/*
		Add the points pers
			One point for every alphanumeric character in the retailer name.
			5 points for every two items on the receipt.
	*/
	points += len(regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(receipt.Retailer, "")) * VALUE_PER_ALPHANUMERIC_CHAR
	points += (len(receipt.Items) / 2) * VALUE_PER_TWO_ITEMS

	/*
		Add the points bonuses
			50 points if the total is a round dollar amount with no cents.
			25 points if the total is a multiple of `0.25`.
			6 points if the day in the purchase date is odd.
			10 points if the time of purchase is after 2:00pm and before 4:00pm.
	*/
	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err == nil && math.Mod(total, 1) == 0 {
		points += ROUND_DOLLAR_AMOUNT_BONUS
	}
	if err == nil && math.Mod(total, 0.25) == 0 {
		points += MULTIPLE_OF_0_POINT_25_BONUS
	}
	day, err := strconv.Atoi(strings.Split(receipt.PurchaseDate, "-")[2])
	if err == nil && day%2 == 1 {
		points += ODD_DAY_BONUS
	}
	hour, err := strconv.Atoi(strings.Split(receipt.PurchaseTime, ":")[0])
	if err == nil && hour >= 14 && hour < 16 {
		points += BETWEEN_2PM_AND_4PM_BONUS
	}

	/*
		Add the value of each item
	*/
	for i := 0; i < len(receipt.Items); i++ {
		if len(strings.TrimSpace(receipt.Items[i].ShortDescription))%3 == 0 {
			price, err := strconv.ParseFloat(receipt.Items[i].Price, 64)
			if err == nil {
				points += int(math.Ceil(price * ITEM_PRICE_MULTIPLIER))
			}
		}
	}

	// return the points as a json object with a 200 status
	context.JSON(http.StatusOK, Points{Points: points})
}
